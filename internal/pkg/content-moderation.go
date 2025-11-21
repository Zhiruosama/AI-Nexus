// Package pkg 内容审核工具
package pkg

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
)

// DFANode DFA字典树节点
type DFANode struct {
	children map[rune]*DFANode
	isEnd    bool
}

// ContentModerator 内容审核器
type ContentModerator struct {
	root *DFANode
}

var (
	moderator *ContentModerator
)

// ValidatePrompt 验证 prompt 内容
func ValidatePrompt(prompt string) error {
	if err := validateContent(prompt); err != nil {
		return fmt.Errorf("prompt validation failed: %w", err)
	}

	return nil
}

func init() {
	moderator = &ContentModerator{
		root: &DFANode{
			children: make(map[rune]*DFANode),
			isEnd:    false,
		},
	}

	if err := moderator.loadFromFile("configs/prohibited-words.txt"); err != nil {
		panic("The configs/prohibited-words.txt doesn't exist")
	}
}

// loadFromFile 从文件加载违禁词库
func (cm *ContentModerator) loadFromFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer func() {
		errs := file.Close()
		if errs != nil {
			logger.Error(nil, "Close file error: %s", errs.Error())
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cm.addWord(line)
	}

	return scanner.Err()
}

// addWord 添加单个违禁词到DFA树
func (cm *ContentModerator) addWord(word string) {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return
	}

	node := cm.root
	for _, r := range word {
		if node.children[r] == nil {
			node.children[r] = &DFANode{
				children: make(map[rune]*DFANode),
			}
		}
		node = node.children[r]
	}
	node.isEnd = true
}

// validateContent 验证内容是否合规
func validateContent(text string) error {
	hasProhibited, words := moderator.check(text)

	if hasProhibited {
		return fmt.Errorf("content contains prohibited words: %s", strings.Join(words, ", "))
	}

	return nil
}

// check 检查文本是否包含违禁词
func (cm *ContentModerator) check(text string) (bool, []string) {
	text = strings.ToLower(text)
	runes := []rune(text)
	matches := make([]string, 0)
	matchSet := make(map[string]bool)

	for i := range len(runes) {
		node := cm.root
		j := i
		matchedWord := make([]rune, 0)

		for j < len(runes) {
			r := runes[j]
			if node.children[r] == nil {
				break
			}
			node = node.children[r]
			matchedWord = append(matchedWord, r)

			if node.isEnd {
				word := string(matchedWord)
				if !matchSet[word] {
					matches = append(matches, word)
					matchSet[word] = true
				}
			}
			j++
		}
	}

	return len(matches) > 0, matches
}
