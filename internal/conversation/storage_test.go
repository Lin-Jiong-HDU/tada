package conversation

import (
	"os"
	"testing"
	"time"
)

func TestFileStorage_SaveAndGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "conv-storage-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewFileStorage(tmpDir)

	conv := NewConversation("default")
	conv.ID = "test-id-123"
	conv.AddMessage(Message{
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Now(),
	})

	// 保存
	err = storage.Save(conv)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 读取
	loaded, err := storage.Get("test-id-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if loaded.ID != conv.ID {
		t.Errorf("Expected ID %s, got %s", conv.ID, loaded.ID)
	}

	if len(loaded.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(loaded.Messages))
	}
}

func TestFileStorage_List(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "conv-storage-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewFileStorage(tmpDir)

	// 创建多个对话
	conv1 := NewConversation("default")
	conv1.ID = "id-1"
	storage.Save(conv1)

	conv2 := NewConversation("coder")
	conv2.ID = "id-2"
	storage.Save(conv2)

	// 列出
	list, err := storage.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(list))
	}
}

func TestFileStorage_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "conv-storage-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewFileStorage(tmpDir)

	conv := NewConversation("default")
	conv.ID = "test-id"
	storage.Save(conv)

	// 删除
	err = storage.Delete("test-id")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证已删除
	_, err = storage.Get("test-id")
	if err == nil {
		t.Error("Expected error when getting deleted conversation")
	}
}
