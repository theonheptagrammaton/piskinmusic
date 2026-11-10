//go:build linux

package visualizer

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

const (
	pulseSampleRate          = 48_000
	pulseLatencySeconds      = 0.03
	pulseQueueDepth          = 8
	pulseClientTimeout       = 3 * time.Second
	pulseServerProbeInterval = time.Second
)

func openSystemSource() (Source, error) {
	client, err := pulse.NewClient(
		pulse.ClientApplicationName("amtui"),
		pulse.ClientTimeout(pulseClientTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to PulseAudio server: %w", err)
	}
	var serverInfo proto.GetServerInfoReply
	if err := client.RawRequest(&proto.GetServerInfo{}, &serverInfo); err != nil {
		client.Close()
		return nil, fmt.Errorf("identify PulseAudio server: %w", err)
	}

	sink, err := client.DefaultSink()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("resolve default PulseAudio sink: %w", err)
	}

	var target atomic.Pointer[pcmSource]
	stream, err := client.NewRecord(
		pulse.Float32Writer(func(samples []float32) (int, error) {
			source := target.Load()
			if source == nil {
				return len(samples), nil
			}
			return source.write(samples)
		}),
		pulse.RecordMonitor(sink),
		pulse.RecordStereo,
		pulse.RecordSampleRate(pulseSampleRate),
		pulse.RecordLatency(pulseLatencySeconds),
		pulse.RecordMediaName("amtui system audio visualizer"),
	)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("create PulseAudio monitor stream: %w", err)
	}

	format := Format{
		SampleRate: stream.SampleRate(),
		Channels:   stream.Channels(),
	}
	if err := validateSourceFormat(format); err != nil {
		client.Close()
		return nil, fmt.Errorf("PulseAudio monitor stream format: %w", err)
	}

	monitorDone := make(chan struct{})
	cleanup := newSourceCleanupCoordinator(
		monitorDone,
		client.Close,
	)
	source, err := newPCMSource(pulseSourceName(serverInfo.PackageName), format, pulseQueueDepth, cleanup.close)
	if err != nil {
		client.Close()
		return nil, err
	}
	target.Store(source)

	// RecordStream.Start mutates unsynchronized library state also written by
	// its disconnect callback. We own capture lifetime via client.Close and
	// never use the stream's state helpers; send the same protocol requests
	// directly, checking failures before publishing a usable source.
	if err := startPulseRecord(client, stream.StreamIndex()); err != nil {
		source.fail(err)
		close(monitorDone) // no server monitor was started
		_ = source.Close()
		return nil, fmt.Errorf("start PulseAudio monitor stream: %w", err)
	}
	go monitorPulseServer(source, client, monitorDone)
	return source, nil
}

type pulseRequester interface {
	RawRequest(proto.RequestArgs, proto.Reply) error
}

func startPulseRecord(client pulseRequester, index uint32) error {
	if err := client.RawRequest(&proto.FlushRecordStream{StreamIndex: index}, nil); err != nil {
		return err
	}
	return client.RawRequest(&proto.CorkRecordStream{StreamIndex: index, Corked: false}, nil)
}

// Use the server's identity, not the presence of a PulseAudio socket, to claim
// PipeWire. For other servers the label describes the protocol we actually use.
func pulseSourceName(serverName string) string {
	if serverName == "PipeWire" || strings.HasPrefix(serverName, "PipeWire ") ||
		strings.HasPrefix(serverName, "PulseAudio (on PipeWire ") {
		return "PIPEWIRE"
	}
	return "PULSEAUDIO"
}

func monitorPulseServer(
	source *pcmSource,
	client *pulse.Client,
	monitorDone chan<- struct{},
) {
	defer close(monitorDone)

	timer := time.NewTimer(pulseServerProbeInterval)
	defer timer.Stop()

	for {
		select {
		case <-source.done:
			return
		case <-timer.C:
		}

		var serverInfo proto.GetServerInfoReply
		if err := client.RawRequest(&proto.GetServerInfo{}, &serverInfo); err != nil {
			source.fail(fmt.Errorf("PulseAudio server connection: %w", err))
			return
		}
		timer.Reset(pulseServerProbeInterval)
	}
}
