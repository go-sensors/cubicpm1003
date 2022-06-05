package cubicpm1003_test

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/go-sensors/core/io/mocks"
	"github.com/go-sensors/core/pm"
	"github.com/go-sensors/core/units"
	"github.com/go-sensors/cubicpm1003"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func Test_NewSensor_returns_a_configured_sensor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	// Act
	sensor := cubicpm1003.NewSensor(portFactory)

	// Assert
	assert.NotNil(t, sensor)
	assert.Equal(t, cubicpm1003.DefaultMeasurementInterval, sensor.MeasurementInterval())
	assert.Equal(t, cubicpm1003.DefaultReconnectTimeout, sensor.ReconnectTimeout())
	assert.Nil(t, sensor.RecoverableErrorHandler())
}

func Test_NewSensor_with_options_returns_a_configured_sensor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	expectedMeasurementInterval := cubicpm1003.DefaultMeasurementInterval * 5
	expectedReconnectTimeout := cubicpm1003.DefaultReconnectTimeout * 10

	// Act
	sensor := cubicpm1003.NewSensor(portFactory,
		cubicpm1003.WithMeasurementInterval(expectedMeasurementInterval),
		cubicpm1003.WithReconnectTimeout(expectedReconnectTimeout),
		cubicpm1003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	// Assert
	assert.NotNil(t, sensor)
	assert.Equal(t, expectedMeasurementInterval, sensor.MeasurementInterval())
	assert.Equal(t, expectedReconnectTimeout, sensor.ReconnectTimeout())
	assert.NotNil(t, sensor.RecoverableErrorHandler())
	assert.True(t, sensor.RecoverableErrorHandler()(nil))
}

func Test_ConcentrationSpecs_returns_supported_concentrations(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	sensor := cubicpm1003.NewSensor(portFactory)
	expected := []*pm.ConcentrationSpec{
		{
			UpperBoundSize:   cubicpm1003.PM2_5UpperBoundSize,
			Resolution:       1 * units.MicrogramPerCubicMeter,
			MinConcentration: 0 * units.MicrogramPerCubicMeter,
			MaxConcentration: 500 * units.MicrogramPerCubicMeter,
		},
	}

	// Act
	actual := sensor.ConcentrationSpecs()

	// Assert
	assert.NotNil(t, actual)
	assert.EqualValues(t, expected, actual)
}

func Test_Run_fails_when_opening_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(nil, errors.New("boom"))
	sensor := cubicpm1003.NewSensor(portFactory)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to open port")
}

func Test_Run_fails_writing_to_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Write(gomock.Any()).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Read(gomock.Any()).
		AnyTimes()
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := cubicpm1003.NewSensor(portFactory,
		cubicpm1003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to write")
}

func Test_Run_fails_reading_header_from_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Write(gomock.Any()).
		Return(0, nil).
		AnyTimes()
	port.EXPECT().
		Read(gomock.Any()).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := cubicpm1003.NewSensor(portFactory,
		cubicpm1003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to read while seeking measurement header")
}

func Test_Run_fails_reading_EOF_from_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Write(gomock.Any()).
		Return(0, nil).
		AnyTimes()
	port.EXPECT().
		Read(gomock.Any()).
		Return(0, io.EOF)
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := cubicpm1003.NewSensor(portFactory,
		cubicpm1003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorIs(t, err, io.EOF)
}

func Test_Run_fails_reading_measurement_from_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Write(gomock.Any()).
		Return(0, nil).
		AnyTimes()
	returnsHeader := port.EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			noise := make([]byte, len(buf)-len(cubicpm1003.ReadingHeader))
			rand.Read(noise)

			copy(buf, append(noise, cubicpm1003.ReadingHeader...))
			return len(buf), nil
		})
	port.EXPECT().
		Read(gomock.Any()).
		Return(0, errors.New("boom")).
		After(returnsHeader)
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := cubicpm1003.NewSensor(portFactory,
		cubicpm1003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to read measurement")
}

func Test_Run_successfully_reads_measurement_from_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Write(gomock.Any()).
		Return(0, nil).
		AnyTimes()
	expected := &pm.Concentration{
		UpperBoundSize: cubicpm1003.PM2_5UpperBoundSize,
		Amount:         420 * units.MicrogramPerCubicMeter,
	}
	port.EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			valid := append(
				cubicpm1003.ReadingHeader,
				[]byte{
					0x00, 0x00, 0x01, 0xA4,
					0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00,
					0x00,
				}...,
			)
			copy(buf, valid)
			return len(valid), nil
		}).
		AnyTimes()
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := cubicpm1003.NewSensor(portFactory,
		cubicpm1003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	var actual *pm.Concentration
	group.Go(func() error {
		select {
		case <-ctx.Done():
			return errors.New("failed to receive expected measurement")
		case actual = <-sensor.Concentrations():
			cancel()
			return nil
		}
	})
	err := group.Wait()

	// Assert
	assert.Nil(t, err)
	assert.EqualValues(t, actual, expected)
}

func Test_Run_attempts_to_recover_from_failure(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Write(gomock.Any()).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Read(gomock.Any()).
		AnyTimes()
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := cubicpm1003.NewSensor(portFactory,
		cubicpm1003.WithRecoverableErrorHandler(func(err error) bool { return false }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.Nil(t, err)
}
