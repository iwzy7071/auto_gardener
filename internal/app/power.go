package app

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type PowerStatus struct {
	Platform  string   `json:"platform"`
	OK        bool     `json:"ok"`
	Checked   bool     `json:"checked"`
	Warnings  []string `json:"warnings,omitempty"`
	Advice    []string `json:"advice,omitempty"`
	CheckedAt string   `json:"checkedAt"`
}

func CheckPowerStatus() PowerStatus {
	ps := PowerStatus{Platform: runtime.GOOS, OK: true, Checked: true, CheckedAt: time.Now().Format(time.RFC3339)}
	switch runtime.GOOS {
	case "windows":
		return checkWindowsPower(ps)
	case "darwin":
		return checkMacPower(ps)
	default:
		ps.Checked = false
		ps.OK = true
		ps.Advice = []string{"当前系统暂不支持自动电源配置检测。远程使用 Gardener 时，请手动确认电脑不会睡眠、休眠或关机。"}
		return ps
	}
}

const maxPowerWarningsTextRunes = 1000

func PowerWarningsText(ps PowerStatus) string {
	if len(ps.Warnings) == 0 && len(ps.Advice) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines, ps.Warnings...)
	lines = append(lines, ps.Advice...)
	text := strings.Join(lines, "\n")
	chars := []rune(strings.TrimSpace(text))
	if len(chars) <= maxPowerWarningsTextRunes {
		return string(chars)
	}
	return string(chars[:maxPowerWarningsTextRunes]) + "…"
}

func checkWindowsPower(ps PowerStatus) PowerStatus {
	checks := []struct{ alias, label string }{
		{"STANDBYIDLE", "睡眠"},
		{"HIBERNATEIDLE", "休眠"},
	}
	for _, c := range checks {
		out, err := exec.Command("powercfg", "/query", "SCHEME_CURRENT", "SUB_SLEEP", c.alias).CombinedOutput()
		if err != nil {
			ps.OK = false
			ps.Warnings = append(ps.Warnings, fmt.Sprintf("无法检测 Windows %s设置：%v", c.label, err))
			continue
		}
		ac, dc := parsePowerCfgIndexes(string(out))
		if ac > 0 {
			ps.OK = false
			ps.Warnings = append(ps.Warnings, fmt.Sprintf("Windows 当前电源计划在接通电源时会自动%s（%d 秒后）。", c.label, ac))
		}
		if dc > 0 {
			ps.OK = false
			ps.Warnings = append(ps.Warnings, fmt.Sprintf("Windows 当前电源计划在使用电池时会自动%s（%d 秒后）。", c.label, dc))
		}
	}
	out, err := exec.Command("powercfg", "/query", "SCHEME_CURRENT", "SUB_BUTTONS", "LIDACTION").CombinedOutput()
	if err == nil {
		ac, dc := parsePowerCfgIndexes(string(out))
		if ac != 0 || dc != 0 {
			ps.OK = false
			ps.Warnings = append(ps.Warnings, "Windows 合盖动作可能会让电脑睡眠/休眠/关机，远程任务期间请不要合盖，或在电源设置中改为“不采取任何操作”。")
		}
	}
	if !ps.OK {
		ps.Advice = append(ps.Advice,
			"建议：设置 → 系统 → 电源和电池，将“屏幕和睡眠/睡眠”全部改为“从不”。",
			"也可以用管理员 PowerShell 执行：powercfg /change standby-timeout-ac 0; powercfg /change standby-timeout-dc 0; powercfg /change hibernate-timeout-ac 0; powercfg /change hibernate-timeout-dc 0",
			"远程访问 Gardener 期间电脑必须保持开机、联网，不能关机；系统无法阻止用户主动关机，只能提前提示。",
		)
	}
	return ps
}

func parsePowerCfgIndexes(out string) (ac, dc int64) {
	reAC := regexp.MustCompile(`(?i)Current AC Power Setting Index:\s*0x([0-9a-f]+)`)
	reDC := regexp.MustCompile(`(?i)Current DC Power Setting Index:\s*0x([0-9a-f]+)`)
	if m := reAC.FindStringSubmatch(out); len(m) == 2 {
		ac, _ = strconv.ParseInt(m[1], 16, 64)
	}
	if m := reDC.FindStringSubmatch(out); len(m) == 2 {
		dc, _ = strconv.ParseInt(m[1], 16, 64)
	}
	return ac, dc
}

func checkMacPower(ps PowerStatus) PowerStatus {
	out, err := exec.Command("pmset", "-g", "custom").CombinedOutput()
	if err != nil {
		out, err = exec.Command("pmset", "-g").CombinedOutput()
	}
	if err != nil {
		ps.OK = false
		ps.Warnings = append(ps.Warnings, "无法检测 macOS 电源设置："+err.Error())
		ps.Advice = append(ps.Advice, "请手动打开 系统设置 → 电池/锁定屏幕，把自动睡眠相关选项设为永不。")
		return ps
	}
	values := parsePMSetValues(string(out))
	for profile, kv := range values {
		if v, ok := kv["sleep"]; ok && v > 0 {
			ps.OK = false
			ps.Warnings = append(ps.Warnings, fmt.Sprintf("macOS 在 %s 配置下会在 %d 分钟后睡眠。", profile, v))
		}
		if v, ok := kv["standby"]; ok && v > 0 {
			ps.OK = false
			ps.Warnings = append(ps.Warnings, fmt.Sprintf("macOS 在 %s 配置下启用了 standby，长时间无人值守可能断开远程访问。", profile))
		}
		if v, ok := kv["autopoweroff"]; ok && v > 0 {
			ps.OK = false
			ps.Warnings = append(ps.Warnings, fmt.Sprintf("macOS 在 %s 配置下启用了 autopoweroff，长时间运行可能自动断开。", profile))
		}
	}
	if !ps.OK {
		ps.Advice = append(ps.Advice,
			"建议：系统设置 → 电池/锁定屏幕，将自动睡眠设为永不；远程任务期间不要合盖。",
			"也可以执行：sudo pmset -a sleep 0 disksleep 0 standby 0 autopoweroff 0",
			"远程访问 Gardener 期间电脑必须保持开机、联网，不能关机；系统无法阻止用户主动关机，只能提前提示。",
		)
	}
	return ps
}

func parsePMSetValues(out string) map[string]map[string]int64 {
	res := map[string]map[string]int64{}
	profile := "current"
	reProfile := regexp.MustCompile(`^\s*(Battery Power|AC Power|UPS Power|Currently in use):`)
	reKV := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s+(-?\d+)`)
	for _, line := range strings.Split(out, "\n") {
		if m := reProfile.FindStringSubmatch(line); len(m) == 2 {
			profile = m[1]
			if res[profile] == nil {
				res[profile] = map[string]int64{}
			}
			continue
		}
		if m := reKV.FindStringSubmatch(line); len(m) == 3 {
			if res[profile] == nil {
				res[profile] = map[string]int64{}
			}
			v, _ := strconv.ParseInt(m[2], 10, 64)
			res[profile][strings.ToLower(m[1])] = v
		}
	}
	return res
}
