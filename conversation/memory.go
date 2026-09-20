package conversation

import (
	"crypto/rand"
	"regexp"
	"strings"
)

const BotName = "Nyansapo"

type Turn struct {
	Question string
	Answer   string
}

type Memory struct {
	turns []Turn
	limit int
	name  string
}

func NewMemory(limit int) *Memory {
	if limit < 1 {
		limit = 8
	}
	return &Memory{limit: limit}
}

var referenceWords = []string{
	"and", "also", "same", "again", "it", "them", "those", "these",
	"earlier", "previous", "too", "what about", "how about",
}

var namePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bmy name is\s+([a-z][a-z .-]*)`),
	regexp.MustCompile(`(?i)\b(?:i am|i'm)\s+([a-z]{1,24})$`),
	regexp.MustCompile(`(?i)\bcall me\s+([a-z][a-z .-]*)`),
	regexp.MustCompile(`(?i)\bname's\s+([a-z][a-z .-]*)`),
}

func (m *Memory) Respond(question string) (string, bool) {
	question = strings.TrimSpace(question)
	if question == "" {
		return "", false
	}
	lower := strings.ToLower(question)

	greeting := regexp.MustCompile(`^\s*(hi|hey|hello|good (morning|afternoon|evening)|yo)\b`).FindString(lower)
	if greeting != "" && !strings.Contains(lower, "?") {
		return "Hello! I'm " + BotName + ". Ask me about your data, like revenue or attendance.", true
	}

	if regexp.MustCompile(`\b(bye|goodbye|see you)\b`).MatchString(lower) {
		return "Goodbye!", true
	}

	if strings.Contains(lower, "your name") || strings.Contains(lower, "what are you") ||
		(strings.Contains(lower, "who") && strings.Contains(lower, "you")) {
		return "I am " + BotName + ", your read-only business data assistant.", true
	}

	if strings.Contains(lower, "my name") && strings.Contains(lower, "what") {
		if m.name != "" {
			return "Your name is " + m.name + ". I remember that.", true
		}
		return "I don't know your name yet. Tell me, what is your name?", true
	}

	for _, pattern := range namePatterns {
		if match := pattern.FindStringSubmatch(question); len(match) >= 2 {
			name := strings.Trim(match[1], " .-")
			m.name = name
			return "Nice to meet you, " + name + "!", true
		}
	}

	if strings.Contains(lower, "remember") && strings.Contains(lower, "my name") {
		if m.name != "" {
			return "I remember your name: " + m.name + ".", true
		}
		return "You haven't told me your name yet.", true
	}

	if reply, ok := m.smalltalk(lower); ok {
		return reply, true
	}

	return "", false
}

func (m *Memory) smalltalk(lower string) (string, bool) {
	switch {
	case regexp.MustCompile(`\b(i'?m|i am)\b\s+(?:doing\s+)?(?:a (?:bit|little)|so|very|really|quite)\s+(well|great|good|fine|ok|okay|awesome|amazing|happy|tired|stressed|bored|sad|sick|hungry)\b`).MatchString(lower):
		fallthrough
	case regexp.MustCompile(`\b(i'?m|i am)\s+(doing )?(well|great|good|fine|ok|okay|awesome|amazing|happy|tired|stressed|bored|sad|sick|hungry)\b`).MatchString(lower):
		reply, _ := m.feeling(lower)
		return reply, true

	case regexp.MustCompile(`\b(how are you|how are things|how's it going|how r u|what about you|and you)\b`).MatchString(lower):
		return pick(moodTemplates), true

	case regexp.MustCompile(`\b(what would you like to eat|what do you (want to|like to) eat|have you eaten|what do you eat|you ('ve| have) (not|never) eaten|you haven.t eaten)\b`).MatchString(lower):
		return pick(foodTemplates), true

	case regexp.MustCompile(`\b(i love (you|this|it)|you.re (great|awesome|amazing|smart|the best)|thank(s| you))\b`).MatchString(lower):
		return pick(praiseTemplates), true

	case regexp.MustCompile(`\b(are you (a )?(robot|human|real|alive|ai)|do you (have feelings|sleep|eat)\b)`).MatchString(lower):
		return pick(natureTemplates), true

	case regexp.MustCompile(`\b(what do you think|your opinion|do you like)\b`).MatchString(lower):
		return "I don't have real opinions, but I do like a clean query that returns exactly what you need.", true
	}
	return "", false
}

func (m *Memory) feeling(lower string) (string, bool) {
	switch {
	case regexp.MustCompile(`\btired\b`).MatchString(lower):
		return pick(tiredTemplates), true
	case regexp.MustCompile(`\bstressed\b`).MatchString(lower):
		return pick(stressedTemplates), true
	case regexp.MustCompile(`\bsad\b`).MatchString(lower):
		return pick(sadTemplates), true
	case regexp.MustCompile(`\bbored\b`).MatchString(lower):
		return "Bored? Ask me for some interesting numbers from your data, I promise they add up.", true
	case regexp.MustCompile(`\bhappy\b`).MatchString(lower):
		return "That's lovely to hear. Good mood and clear questions make for my favorite kind of day.", true
	case regexp.MustCompile(`\bsick\b`).MatchString(lower):
		return "Feel better soon! I'll keep your data sorted while you rest.", true
	case regexp.MustCompile(`\bhungry\b`).MatchString(lower):
		return "Hungry sounds good. I'd help, but I run on queries rather than curry.", true
	default:
		return pick(wellTemplates), true
	}
}

func (m *Memory) Enrich(question string) string {
	question = strings.TrimSpace(question)
	if question == "" || len(m.turns) == 0 {
		return question
	}
	last := m.turns[len(m.turns)-1].Question
	lower := strings.ToLower(question)
	significant := 0
	for _, word := range strings.Fields(lower) {
		if len(word) > 2 {
			significant++
		}
	}
	references := false
	for _, reference := range referenceWords {
		if strings.Contains(lower, reference) {
			references = true
			break
		}
	}
	if significant <= 3 || references {
		return question + " Context: " + last
	}
	return question
}

func (m *Memory) Repeat(question string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(question))
	for _, phrase := range []string{"say that again", "again", "repeat", "what was"} {
		if strings.Contains(lower, phrase) {
			if len(m.turns) > 0 {
				return m.turns[len(m.turns)-1].Answer, true
			}
			return "", false
		}
	}
	return "", false
}

func (m *Memory) Remember(question, answer string) {
	m.turns = append(m.turns, Turn{Question: strings.TrimSpace(question), Answer: answer})
	if len(m.turns) > m.limit {
		m.turns = m.turns[len(m.turns)-m.limit:]
	}
}

var moodTemplates = []string{
	"I'm doing well, thanks for asking. I'm Nyansapo and always ready to look up your data.",
	"I'm doing great! Keep the questions coming and I'll dig into your numbers.",
	"Feeling sharp and ready. What can I pull from your database today?",
	"All good here. Now, tell me what you'd like to know — revenue, attendance, counts?",
}

var foodTemplates = []string{
	"I'd love a big plate of fresh SQL queries, lightly seasoned with clean data.",
	"I run on data, not food. But a well-indexed table would be a feast.",
	"Maybe a warm bowl of aggregated rows and a side of accurate results.",
	"I don't eat, but I do feast on well-structured data. What about you?",
}

var praiseTemplates = []string{
	"You're welcome! Happy to help with your data.",
	"Thank you! That's kind. I'm your data assistant, so this is my job and joy.",
	"I appreciate that. Ask me anything about your database anytime.",
	"Aww, I'm flattered. Let's dig into some numbers together.",
}

var natureTemplates = []string{
	"I'm not a human — I'm Nyansapo, a read-only business data assistant. But I try to be friendly!",
	"I'm software, not a person. But I enjoy a good conversation and crunching your data.",
	"A mix of both worlds: I feel *nothing* yet I'm here to help. Ask me about your records.",
	"I'm an AI assistant. I don't have feelings, but I'm happy to collaborate on answers.",
}

var tiredTemplates = []string{
	"Get some rest! Your data will still be here when you're back.",
	"That sounds rough. Sleep well — the numbers can wait.",
	"You've earned a break. I'll keep an eye on the database.",
}

var stressedTemplates = []string{
	"Take a breath, one step at a time. Maybe I can lift one thing off your plate — just ask.",
	"Stress is heavy. If any of it involves data, lookups, or reports — point me at it.",
	"Deep breath. I'm here to handle the grunt work so you don't have to.",
}

var sadTemplates = []string{
	"I'm sorry to hear that. If talking it through helps, I'm a good listener.",
	"That stinks. I hope tomorrow treats you better. I'm here if you need to unpack a bit.",
	"Hang in there. Sometimes even the driest data day gets a little brighter.",
}

var wellTemplates = []string{
	"That's great to hear. Let's keep the momentum with a couple of questions!",
	"Glad you're doing well. Anything on your mind data-wise?",
	"Love to hear it. What should we look at next?",
}

func pick(templates []string) string {
	return templates[randomIndex(len(templates))]
}

func randomIndex(count int) int {
	if count < 2 {
		return 0
	}
	var value [1]byte
	if _, err := rand.Read(value[:]); err == nil {
		return int(value[0]) % count
	}
	return 0
}