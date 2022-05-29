package fans

import (
	"errors"
	"fmt"
	"github.com/markusressel/fan2go/internal/configuration"
	"github.com/markusressel/fan2go/internal/ui"
	"github.com/markusressel/fan2go/internal/util"
	"os"
	"path"
	"path/filepath"
)

type HwMonFan struct {
	Label        string                  `json:"label"`
	Index        int                     `json:"index"`
	RpmMovingAvg float64                 `json:"rpmMovingAvg"`
	Config       configuration.FanConfig `json:"config"`
	StartPwm     *int                    `json:"startPwm"` // the min PWM at which the fan starts to rotate from a stand still
	FanCurveData *map[int]float64        `json:"fanCurveData"`
	Rpm          int                     `json:"rpm"`
	Pwm          int                     `json:"pwm"`
}

func (fan HwMonFan) GetId() string {
	return fan.Config.ID
}

func (fan HwMonFan) GetStartPwm() int {
	if fan.StartPwm != nil {
		return *fan.StartPwm
	} else {
		return MaxPwmValue
	}
}

func (fan *HwMonFan) SetStartPwm(pwm int) {
	fan.StartPwm = &pwm
}

func (fan HwMonFan) GetMinPwm() int {
	// if the fan is never supposed to stop,
	// use the lowest pwm value where the fan is still spinning
	if fan.ShouldNeverStop() {
		if len(fan.Config.HwMon.RpmInput) <= 0 {
			ui.Warning("WARN: cannot guarantee neverStop option on fan %s, since it has no RPM input.", fan.GetId())
		}

		var minPwm int
		if fan.Config.MinPwm != nil {
			minPwm = *fan.Config.MinPwm
		} else {
			minPwm = MinPwmValue
		}
		return minPwm
	}

	return MinPwmValue
}

func (fan *HwMonFan) SetMinPwm(pwm int) {
	fan.Config.MinPwm = &pwm
}

func (fan HwMonFan) GetMaxPwm() int {
	var maxPwm int
	if fan.Config.MaxPwm != nil {
		maxPwm = *fan.Config.MaxPwm
	} else {
		maxPwm = MaxPwmValue
	}
	return maxPwm
}

func (fan *HwMonFan) SetMaxPwm(pwm int) {
	fan.Config.MaxPwm = &pwm
}

func (fan *HwMonFan) GetRpm() (int, error) {
	if value, err := util.ReadIntFromFile(fan.Config.HwMon.RpmInput); err != nil {
		return 0, err
	} else {
		fan.Rpm = value
		return value, nil
	}
}

func (fan HwMonFan) GetRpmAvg() float64 {
	return fan.RpmMovingAvg
}

func (fan *HwMonFan) SetRpmAvg(rpm float64) {
	fan.RpmMovingAvg = rpm
}

func (fan *HwMonFan) GetPwm() (int, error) {
	value, err := util.ReadIntFromFile(fan.Config.HwMon.PwmOutput)
	if err != nil {
		return MinPwmValue, err
	}
	fan.Pwm = value
	return value, nil
}

func (fan *HwMonFan) SetPwm(pwm int) (err error) {
	ui.Debug("Setting Fan PWM of '%s' to %d ...", fan.GetId(), pwm)
	err = util.WriteIntToFile(pwm, fan.Config.HwMon.PwmOutput)
	return err
}

func (fan HwMonFan) GetFanCurveData() *map[int]float64 {
	return fan.FanCurveData
}

// AttachFanCurveData attaches fan curve data from persistence to a fan
// Note: When the given data is incomplete, all values up until the highest
// value in the given dataset will be interpolated linearly
// returns os.ErrInvalid if curveData is void of any data
func (fan *HwMonFan) AttachFanCurveData(curveData *map[int]float64) (err error) {
	if curveData == nil || len(*curveData) <= 0 {
		ui.Error("Cant attach empty fan curve data to fan %s", fan.GetId())
		return os.ErrInvalid
	}

	fan.FanCurveData = curveData

	startPwm, maxPwm := ComputePwmBoundaries(fan)
	fan.SetStartPwm(startPwm)
	fan.SetMaxPwm(maxPwm)

	// TODO: we don't have a way to determine this yet
	fan.SetMinPwm(startPwm)

	return err
}

func (fan HwMonFan) GetCurveId() string {
	return fan.Config.Curve
}

func (fan HwMonFan) ShouldNeverStop() bool {
	return fan.Config.NeverStop
}

func (fan HwMonFan) GetPwmEnabled() (int, error) {
	pwmEnabledFilePath := pwmEnablePath(fan)
	return util.ReadIntFromFile(pwmEnabledFilePath)
}

func (fan HwMonFan) IsPwmAuto() (bool, error) {
	value, err := fan.GetPwmEnabled()
	if err != nil {
		return false, err
	}
	return value > 1, nil
}

// SetPwmEnabled writes the given value to pwmX_enable
// Possible values (unsure if these are true for all scenarios):
// 0 - no control (results in max speed)
// 1 - manual pwm control
// 2 - motherboard pwm control
func (fan *HwMonFan) SetPwmEnabled(value ControlMode) (err error) {
	pwmEnabledFilePath := pwmEnablePath(*fan)

	err = util.WriteIntToFile(int(value), pwmEnabledFilePath)
	if err == nil {
		currentValue, err := util.ReadIntFromFile(pwmEnabledFilePath)
		if err != nil || ControlMode(currentValue) != value {
			return errors.New(fmt.Sprintf("PWM mode stuck to %d", currentValue))
		}
	}
	return err
}

func pwmEnablePath(f HwMonFan) string {
	folder, _ := filepath.Split(f.Config.HwMon.PwmOutput)
	return path.Join(folder, fmt.Sprintf("pwm%d_enable", f.Index))
}

func (fan HwMonFan) Supports(feature FeatureFlag) bool {
	switch feature {
	case FeatureControlMode:
		pwmEnableFilePath := pwmEnablePath(fan)
		_, err := os.Stat(pwmEnableFilePath)
		return err == nil
	case FeatureRpmSensor:
		return len(fan.Config.HwMon.RpmInput) > 0
	}
	return false
}
