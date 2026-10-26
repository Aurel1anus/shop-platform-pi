package migrations

import (
	"strings"
	"testing"
)

func TestSchemaSeparatesCommerceFromAgentAndSeeds(t *testing.T) {
	for _, name := range []string{"orders", "payments", "accounts", "product_skus", "stored_files", "knowledge_documents"} {
		if !strings.Contains(Schema, "CREATE TABLE IF NOT EXISTS "+name+" (") {
			t.Errorf("missing %s", name)
		}
	}
	for _, legacy := range []string{"agent_runs", "agent_prompts", "chat_sessions", "user_messages", "knowledge_chunks", "INSERT IGNORE"} {
		if strings.Contains(Schema, legacy) {
			t.Errorf("schema contains %s", legacy)
		}
	}
	if !strings.Contains(Schema, "password_hash VARCHAR(128)") {
		t.Fatal("bcrypt hash storage is too short")
	}
	if strings.Contains(Seed, "'indexed'") || strings.Contains(Seed, "knowledge_chunks") {
		t.Fatal("seed contains precomputed RAG state")
	}
}
