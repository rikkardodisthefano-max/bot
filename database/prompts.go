package database

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

type Prompt struct {
	ID      int64
	GuildID string
	Name    string
	Content string
}

func InitPromptStore() {
	if DB == nil {
		return
	}
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS ai_prompts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		UNIQUE(guild_id, name)
	);`)
	if err != nil {
		log.Println("Erro ao criar tabela de prompts:", err)
	}
}

func AddPrompt(guildID, name, content string) error {
	name = strings.TrimSpace(name)
	content = strings.TrimSpace(content)
	if guildID == "" || name == "" || content == "" {
		return fmt.Errorf("nome e conteúdo são obrigatórios")
	}
	_, err := DB.Exec("INSERT INTO ai_prompts (guild_id, name, content) VALUES (?, ?, ?)", guildID, name, content)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return fmt.Errorf("já existe um prompt chamado %q", name)
		}
		return err
	}
	return nil
}

func EditPrompt(guildID, name, content string) error {
	name = strings.TrimSpace(name)
	content = strings.TrimSpace(content)
	if name == "" || content == "" {
		return fmt.Errorf("nome e conteúdo são obrigatórios")
	}
	result, err := DB.Exec("UPDATE ai_prompts SET content = ? WHERE guild_id = ? AND name = ?", content, guildID, name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("prompt %q não encontrado", name)
	}
	return nil
}

func DeletePrompt(guildID, name string) error {
	name = strings.TrimSpace(name)
	result, err := DB.Exec("DELETE FROM ai_prompts WHERE guild_id = ? AND name = ?", guildID, name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("prompt %q não encontrado", name)
	}
	return nil
}

func ListPrompts(guildID string) ([]Prompt, error) {
	rows, err := DB.Query("SELECT id, guild_id, name, content FROM ai_prompts WHERE guild_id = ?", guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.GuildID, &p.Name, &p.Content); err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(prompts, func(i, j int) bool { return strings.ToLower(prompts[i].Name) < strings.ToLower(prompts[j].Name) })
	return prompts, nil
}

func GetPromptContext(guildID string) string {
	prompts, err := ListPrompts(guildID)
	if err != nil || len(prompts) == 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range prompts {
		b.WriteString("\n[Prompt: ")
		b.WriteString(p.Name)
		b.WriteString("]\n")
		b.WriteString(p.Content)
		b.WriteString("\n")
	}
	return b.String()
}
