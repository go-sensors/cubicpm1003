package cubicpm1003

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"time"

	coreio "github.com/go-sensors/core/io"
	"github.com/go-sensors/core/pm"
	"github.com/go-sensors/core/units"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Sensor represents a configured Cubic PM1003 particulate matter sensor
type Sensor struct {
	concentrations      chan *pm.Concentration
	portFactory         coreio.PortFactory
	measurementInterval time.Duration
	reconnectTimeout    time.Duration
	errorHandlerFunc    ShouldTerminate
}

// Option is a configured option that may be applied to a Sensor
type Option struct {
	apply func(*Sensor)
}

// NewSensor creates a Sensor with optional configuration
func NewSensor(portFactory coreio.PortFactory, options ...*Option) *Sensor {
	concentrations := make(chan *pm.Concentration)
	s := &Sensor{
		concentrations:      concentrations,
		portFactory:         portFactory,
		measurementInterval: DefaultMeasurementInterval,
		reconnectTimeout:    DefaultReconnectTimeout,
	}
	for _, o := range options {
		o.apply(s)
	}
	return s
}

// WithMeasurementInterval specifies the duration to wait between reading measurements
func WithMeasurementInterval(interval time.Duration) *Option {
	return &Option{
		apply: func(s *Sensor) {
			s.measurementInterval = interval
		},
	}
}

// MeasurementInterval is the duration to wait between reading measurements
func (s *Sensor) MeasurementInterval() time.Duration {
	return s.measurementInterval
}

// WithReconnectTimeout specifies the duration to wait before reconnecting after a recoverable error
func WithReconnectTimeout(timeout time.Duration) *Option {
	return &Option{
		apply: func(s *Sensor) {
			s.reconnectTimeout = timeout
		},
	}
}

// ReconnectTimeout is the duration to wait before reconnecting after a recoverable error
func (s *Sensor) ReconnectTimeout() time.Duration {
	return s.reconnectTimeout
}

// ShouldTerminate is a function that returns a result indicating whether the Sensor should terminate after a recoverable error
type ShouldTerminate func(error) bool

// WithRecoverableErrorHandler registers a function that will be called when a recoverable error occurs
func WithRecoverableErrorHandler(f ShouldTerminate) *Option {
	return &Option{
		apply: func(s *Sensor) {
			s.errorHandlerFunc = f
		},
	}
}

// RecoverableErrorHandler a function that will be called when a recoverable error occurs
func (s *Sensor) RecoverableErrorHandler() ShouldTerminate {
	return s.errorHandlerFunc
}

var (
	// Command to write to the sensor to invoke its measurement routine
	ReadMeasurementCommand = []byte{0x11, 0x02, 0x0B, 0x01, 0xE1}
	// Header of a valid measurement
	ReadingHeader = []byte{0x16, 0x11, 0x0B}
)

const (
	PM2_5UpperBoundSize = 2500 * units.Nanometer
)

type measurement struct {
	DF1                byte
	DF2                byte
	PM2_5Concentration uint16
	DF5                byte
	DF6                byte
	DF7                byte
	DF8                byte
	DF9                byte
	DF10               byte
	DF11               byte
	DF12               byte
	DF13               byte
	DF14               byte
	DF15               byte
	DF16               byte
	CS                 byte
}

// Run begins reading from the sensor and blocks until either an error occurs or the context is completed
func (s *Sensor) Run(ctx context.Context) error {
	defer close(s.concentrations)
	for {
		port, err := s.portFactory.Open()
		if err != nil {
			return errors.Wrap(err, "failed to open port")
		}

		group, innerCtx := errgroup.WithContext(ctx)
		group.Go(func() error {
			<-innerCtx.Done()
			return port.Close()
		})
		group.Go(func() error {
			for {
				_, err := port.Write(ReadMeasurementCommand)
				if err != nil {
					return errors.Wrap(err, "failed to write 'read measurement' command")
				}

				select {
				case <-innerCtx.Done():
					return nil
				case <-time.After(s.measurementInterval):
				}
			}
		})
		group.Go(func() error {
			reader := bufio.NewReader(port)
			for {
				nextHeaderIndex := 0
				for {
					b, err := reader.ReadByte()
					if err != nil {
						if err == io.EOF {
							return err
						}
						return errors.Wrap(err, "failed to read while seeking measurement header")
					}

					if b == ReadingHeader[nextHeaderIndex] {
						nextHeaderIndex++
						if nextHeaderIndex == len(ReadingHeader) {
							break
						}
						continue
					}

					nextHeaderIndex = 0
				}

				measurement := &measurement{}
				err = binary.Read(reader, binary.BigEndian, measurement)
				if err != nil {
					return errors.Wrap(err, "failed to read measurement")
				}

				concentration := &pm.Concentration{
					UpperBoundSize: PM2_5UpperBoundSize,
					Amount:         units.MassConcentration(measurement.PM2_5Concentration) * units.MicrogramPerCubicMeter,
				}

				select {
				case <-innerCtx.Done():
					return nil
				case s.concentrations <- concentration:
				}
			}
		})

		err = group.Wait()
		if s.errorHandlerFunc != nil {
			if s.errorHandlerFunc(err) {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.reconnectTimeout):
		}
	}
}

// Concentrations returns a channel of PM concentration readings as they become available from the sensor
func (s *Sensor) Concentrations() chan *pm.Concentration {
	return s.concentrations
}

// ConcentrationSpecs returns a collection of specified measurement ranges supported by the sensor
func (*Sensor) ConcentrationSpecs() []*pm.ConcentrationSpec {
	return []*pm.ConcentrationSpec{
		{
			UpperBoundSize:   PM2_5UpperBoundSize,
			Resolution:       1 * units.MicrogramPerCubicMeter,
			MinConcentration: 0 * units.MicrogramPerCubicMeter,
			MaxConcentration: 500 * units.MicrogramPerCubicMeter,
		},
	}
}
