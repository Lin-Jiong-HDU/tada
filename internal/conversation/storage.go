package conversation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Storage 对话存储接口
type Storage interface {
	Save(conv *Conversation) error
	Get(id string) (*Conversation, error)
	List() ([]*Conversation, error)
	ListToday() ([]*Conversation, error)
	Delete(id string) error
}

// FileStorage 文件系统存储实现
type FileStorage struct {
	conversationsDir string
}

// NewFileStorage 创建 FileStorage
func NewFileStorage(conversationsDir string) *FileStorage {
	return &FileStorage{
		conversationsDir: conversationsDir,
	}
}

// GetDatePath 获取对话的日期路径 (YYYYMMDD)
func (s *FileStorage) GetDatePath(conv *Conversation) string {
	date := conv.CreatedAt.Format("20060102")
	return filepath.Join(s.conversationsDir, date)
}

// GetConversationPath 获取对话的完整路径
func (s *FileStorage) GetConversationPath(convID string) (string, error) {
	// 遍历日期文件夹查找
	entries, err := os.ReadDir(s.conversationsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read conversations directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		convPath := filepath.Join(s.conversationsDir, entry.Name(), convID)
		if _, err := os.Stat(convPath); err == nil {
			return convPath, nil
		}
	}

	return "", fmt.Errorf("conversation not found: %s", convID)
}

// Save 保存对话
func (s *FileStorage) Save(conv *Conversation) error {
	datePath := s.GetDatePath(conv)

	// 创建日期目录
	if err := os.MkdirAll(datePath, 0755); err != nil {
		return fmt.Errorf("failed to create date directory: %w", err)
	}

	convPath := filepath.Join(datePath, conv.ID)

	// 创建对话目录
	if err := os.MkdirAll(convPath, 0755); err != nil {
		return fmt.Errorf("failed to create conversation directory: %w", err)
	}

	// 写入 messages.json
	messagesFile := filepath.Join(convPath, "messages.json")
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	if err := os.WriteFile(messagesFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write messages file: %w", err)
	}

	return nil
}

// Get 获取对话
func (s *FileStorage) Get(id string) (*Conversation, error) {
	convPath, err := s.GetConversationPath(id)
	if err != nil {
		return nil, err
	}

	messagesFile := filepath.Join(convPath, "messages.json")
	data, err := os.ReadFile(messagesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read messages file: %w", err)
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation: %w", err)
	}

	return &conv, nil
}

// List 列出所有对话
func (s *FileStorage) List() ([]*Conversation, error) {
	var conversations []*Conversation

	entries, err := os.ReadDir(s.conversationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return conversations, nil // 目录不存在，返回空列表
		}
		return nil, fmt.Errorf("failed to read conversations directory: %w", err)
	}

	for _, dateEntry := range entries {
		if !dateEntry.IsDir() {
			continue
		}

		datePath := filepath.Join(s.conversationsDir, dateEntry.Name())
		convEntries, err := os.ReadDir(datePath)
		if err != nil {
			continue
		}

		for _, convEntry := range convEntries {
			if !convEntry.IsDir() {
				continue
			}

			conv, err := s.Get(convEntry.Name())
			if err != nil {
				continue
			}

			conversations = append(conversations, conv)
		}
	}

	// 按更新时间排序（最新的在前）
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
	})

	return conversations, nil
}

// ListToday 列出今天的对话
func (s *FileStorage) ListToday() ([]*Conversation, error) {
	var conversations []*Conversation

	// 获取今天的日期路径
	todayPath := filepath.Join(s.conversationsDir, time.Now().Format("20060102"))

	entries, err := os.ReadDir(todayPath)
	if err != nil {
		if os.IsNotExist(err) {
			return conversations, nil // 今天还没有对话
		}
		return nil, fmt.Errorf("failed to read today's conversations directory: %w", err)
	}

	for _, convEntry := range entries {
		if !convEntry.IsDir() {
			continue
		}

		conv, err := s.Get(convEntry.Name())
		if err != nil {
			continue
		}

		conversations = append(conversations, conv)
	}

	// 按更新时间排序（最新的在前）
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
	})

	return conversations, nil
}

// Delete 删除对话
func (s *FileStorage) Delete(id string) error {
	convPath, err := s.GetConversationPath(id)
	if err != nil {
		return err
	}

	return os.RemoveAll(convPath)
}
