package app

import (
	"os"
	"strings"
)

type AcceptanceReport struct {
	Status    string           `json:"status"`
	Score     int              `json:"score"`
	Summary   string           `json:"summary"`
	Checklist []AcceptanceItem `json:"checklist"`
	Risks     []string         `json:"risks,omitempty"`
	NextSteps []string         `json:"nextSteps,omitempty"`
}

type AcceptanceItem struct {
	Label    string `json:"label"`
	Status   string `json:"status"`
	Evidence string `json:"evidence,omitempty"`
}

func buildAcceptanceReport(task *Task) AcceptanceReport {
	if task == nil {
		return AcceptanceReport{Status: "pending", Score: 0, Summary: "任务尚未开始验收。"}
	}
	validationText := latestValidationReport(task)
	allText := strings.ToLower(validationText + "\n" + latestTreeReport(task))
	items := []AcceptanceItem{
		{Label: "目标完成", Status: inferChecklistStatus(allText, []string{"goal status: complete", "complete", "完成", "passed", "通过"}, []string{"partial", "blocked", "failed", "失败", "阻塞"}), Evidence: firstMatchingLine(validationText, "goal status", "目标", "完成", "complete", "partial", "blocked")},
		{Label: "验证通过", Status: inferChecklistStatus(strings.ToLower(validationText), []string{"validation passed", "passed", "通过", "无失败"}, []string{"validation failed", "failed", "失败", "error"}), Evidence: firstMatchingLine(validationText, "validation", "验证", "passed", "failed", "通过", "失败")},
		{Label: "有交付产出", Status: hasDeliverableStatus(task), Evidence: deliverableEvidence(task)},
		{Label: "风险已说明", Status: riskDisclosureStatus(allText), Evidence: firstMatchingLine(validationText, "risk", "风险", "known issue", "limitation")},
	}
	score := scoreAcceptance(items)
	status := "passed"
	for _, item := range items {
		if item.Status == "failed" {
			status = "blocked"
			break
		}
		if item.Status == "warning" && status == "passed" {
			status = "partial"
		}
	}
	if task.Status != StatusFinished {
		status = "pending"
		if score > 80 {
			score = 80
		}
	}
	return AcceptanceReport{
		Status:    status,
		Score:     score,
		Summary:   acceptanceSummary(status, score),
		Checklist: items,
		Risks:     collectLines(validationText, []string{"risk", "风险", "known issue", "limitation", "blocked", "失败"}, 3),
		NextSteps: collectLines(validationText, []string{"next", "下一步", "todo", "建议", "follow-up"}, 3),
	}
}

func latestValidationReport(task *Task) string {
	for i := len(task.Trees) - 1; i >= 0; i-- {
		tr := task.Trees[i]
		if tr != nil && tr.IsValidation && tr.FruitPath != "" {
			return readSmallReport(tr.FruitPath)
		}
	}
	return ""
}

func latestTreeReport(task *Task) string {
	for i := len(task.Trees) - 1; i >= 0; i-- {
		tr := task.Trees[i]
		if tr != nil && !tr.IsValidation && tr.FruitPath != "" {
			return readSmallReport(tr.FruitPath)
		}
	}
	return ""
}

func readSmallReport(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(b) > 64*1024 {
		b = b[:64*1024]
	}
	return string(b)
}

func inferChecklistStatus(text string, positive, negative []string) string {
	if strings.TrimSpace(text) == "" {
		return "warning"
	}
	for _, word := range negative {
		if strings.Contains(text, strings.ToLower(word)) {
			return "failed"
		}
	}
	for _, word := range positive {
		if strings.Contains(text, strings.ToLower(word)) {
			return "passed"
		}
	}
	return "warning"
}

func hasDeliverableStatus(task *Task) string {
	for _, tr := range task.Trees {
		if tr != nil && !tr.IsValidation && tr.FruitPath != "" {
			return "passed"
		}
	}
	return "warning"
}

func deliverableEvidence(task *Task) string {
	count := 0
	for _, tr := range task.Trees {
		if tr != nil && !tr.IsValidation && tr.FruitPath != "" {
			count++
		}
	}
	if count == 0 {
		return "尚未发现子任务报告。"
	}
	return "已发现子任务报告。"
}

func riskDisclosureStatus(text string) string {
	if strings.Contains(text, "risk") || strings.Contains(text, "风险") || strings.Contains(text, "limitation") || strings.Contains(text, "known issue") {
		return "passed"
	}
	return "warning"
}

func scoreAcceptance(items []AcceptanceItem) int {
	if len(items) == 0 {
		return 0
	}
	score := 0
	for _, item := range items {
		switch item.Status {
		case "passed":
			score += 25
		case "warning":
			score += 12
		}
	}
	if score > 100 {
		return 100
	}
	return score
}

func acceptanceSummary(status string, score int) string {
	switch status {
	case "passed":
		return "验收信号良好，可以查看报告并按需继续迭代。"
	case "partial":
		return "已有部分验收信号，但仍有信息不足或潜在风险。"
	case "blocked":
		return "验收发现失败或阻塞信号，建议先查看验证报告。"
	default:
		return "任务仍在进行或尚无完整验收报告。"
	}
}

func firstMatchingLine(text string, needles ...string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(strings.Trim(line, "#-* `\t"))
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		for _, needle := range needles {
			if strings.Contains(lower, strings.ToLower(needle)) {
				return truncateRunes(trimmed, 160)
			}
		}
	}
	return ""
}

func collectLines(text string, needles []string, limit int) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(strings.Trim(line, "#-* `\t"))
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		for _, needle := range needles {
			if strings.Contains(lower, strings.ToLower(needle)) && !seen[trimmed] {
				seen[trimmed] = true
				out = append(out, truncateRunes(trimmed, 160))
				break
			}
		}
		if len(out) >= limit {
			break
		}
	}
	return out
}

func truncateRunes(value string, max int) string {
	r := []rune(value)
	if len(r) <= max {
		return value
	}
	return string(r[:max]) + "..."
}
