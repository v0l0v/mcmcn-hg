package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// Configuración básica
const (
	OutputDir = "hugo-site/content/posts"
	FeedsFile = "feeds.txt"
)

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

var apiKey string

func main() {
	fmt.Println("Iniciando Micomicona Fetcher (Universal Images + AI)...")

	// Leer API Key
	apiKey = getAPIKey()
	if apiKey == "" {
		fmt.Println("⚠️ Advertencia: No se encontró GEMINI_API_KEY en .env. El modo AI estará desactivado.")
	}

	feedsList, err := readFeeds(FeedsFile)
	if err != nil {
		log.Fatalf("Error leyendo archivo de feeds (%s): %v", FeedsFile, err)
	}

	if err := os.MkdirAll(OutputDir, 0755); err != nil {
		log.Fatalf("Error creando directorio de salida: %v", err)
	}

	fp := gofeed.NewParser()

	for _, url := range feedsList {
		if url == "" {
			continue
		}
		fmt.Printf("Procesando feed: %s\n", url)
		feed, err := fp.ParseURL(url)
		if err != nil {
			log.Printf("Error procesando %s: %v", url, err)
			continue
		}

		for _, item := range feed.Items {
			generateMarkdown(feed.Title, item)
		}
	}

	fmt.Println("Fetcher finalizado.")
}

func getAPIKey() string {
	data, _ := os.ReadFile(".env")
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "GEMINI_API_KEY=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "GEMINI_API_KEY="))
		}
	}
	return ""
}

func readFeeds(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var feeds []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			feeds = append(feeds, line)
		}
	}
	return feeds, scanner.Err()
}

func extractImage(item *gofeed.Item) string {
	if media, ok := item.Extensions["media"]; ok {
		if content, ok := media["content"]; ok && len(content) > 0 {
			if url, ok := content[0].Attrs["url"]; ok {
				return url
			}
		}
	}
	for _, enc := range item.Enclosures {
		if strings.HasPrefix(enc.Type, "image/") {
			return enc.URL
		}
	}
	re := regexp.MustCompile(`<img[^>]+src="([^">]+)"`)
	matches := re.FindStringSubmatch(item.Description)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func escapeYAML(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func processWithAI(title, description string) (summary string, isAgenda bool) {
	if apiKey == "" {
		return "", false
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-flash-latest:generateContent?key=" + apiKey
	
	prompt := fmt.Sprintf(`Analiza esta noticia y responde EXACTAMENTE con este formato:
RESUMEN: [Resumen de máximo 15 palabras]
TIPO: [Noticia o Agenda]

Título: %s
Contenido: %s`, title, description)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error llamando a Gemini: %v\n", err)
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Gemini respondió con error HTTP %d\n", resp.StatusCode)
		return "", false
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", false
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		text := geminiResp.Candidates[0].Content.Parts[0].Text
		fmt.Printf("AI Analizando: %s -> Respondió: %s\n", title, strings.ReplaceAll(text, "\n", " "))
		
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Limpiar posibles negritas de la IA
			cleanLine := strings.ReplaceAll(line, "**", "")
			
			if strings.HasPrefix(strings.ToUpper(cleanLine), "RESUMEN:") {
				summary = strings.TrimSpace(cleanLine[8:])
			}
			if strings.HasPrefix(strings.ToUpper(cleanLine), "TIPO:") {
				isAgenda = strings.Contains(strings.ToLower(cleanLine), "agenda")
			}
		}
	}

	return summary, isAgenda
}

func generateMarkdown(sourceName string, item *gofeed.Item) {
	fmt.Printf("Analizando item: %s\n", item.Title)
	// Limpiar título para el nombre de archivo
	safeTitle := strings.ReplaceAll(strings.ToLower(item.Title), " ", "-")
	reg := strings.NewReplacer("?", "", ":", "", "!", "", "\"", "", "'", "", "(", "", ")", "", "[", "", "]", "", ",", "", ".", "", "/", "-")
	safeTitle = reg.Replace(safeTitle)

	runes := []rune(safeTitle)
	if len(runes) > 60 {
		safeTitle = string(runes[:60])
	}

	fileName := fmt.Sprintf("%s.md", safeTitle)
	filePath := filepath.Join(OutputDir, fileName)

	// SI YA EXISTE, NO HACEMOS NADA (Para no gastar API ni duplicar)
	if _, err := os.Stat(filePath); err == nil {
		return
	}

	imageURL := extractImage(item)

	pubDate := time.Now()
	if item.PublishedParsed != nil {
		pubDate = *item.PublishedParsed
	}

	cleanDescription := regexp.MustCompile("<[^>]*>").ReplaceAllString(item.Description, "")
	cleanDescription = strings.TrimSpace(cleanDescription)

	// PROCESAR CON IA (Solo si la noticia es de las últimas 48 horas para ir más rápido)
	summary := ""
	isAgenda := false
	if time.Since(pubDate).Hours() < 48 {
		summary, isAgenda = processWithAI(item.Title, cleanDescription)
		time.Sleep(6 * time.Second) // Pausa solo si usamos la IA
	} else {
		// Para noticias antiguas, usamos los primeros 100 caracteres como resumen básico
		descRunes := []rune(cleanDescription)
		if len(descRunes) > 120 {
			summary = string(descRunes[:120]) + "..."
		} else {
			summary = cleanDescription
		}
	}
	
	postType := "Noticia"
	if isAgenda {
		postType = "Agenda"
	}

	// Extraer Hashtags del texto
	hashtagRegex := regexp.MustCompile(`#(\w+)`)
	hashtagMatches := hashtagRegex.FindAllStringSubmatch(cleanDescription, -1)
	extractedTags := []string{postType} 
	for _, m := range hashtagMatches {
		if len(m) > 1 {
			tag := strings.Title(strings.ToLower(m[1]))
			extractedTags = append(extractedTags, tag)
		}
	}
	// Eliminar duplicados
	uniqueTags := make(map[string]bool)
	var finalTags []string
	for _, t := range extractedTags {
		if !uniqueTags[t] {
			uniqueTags[t] = true
			finalTags = append(finalTags, t)
		}
	}
	tagsStr := strings.Join(finalTags, "\", \"")

	content := fmt.Sprintf(`---
title: "%s"
date: %s
categories: ["%s"]
tags: ["%s"]
summary: "%s"
featured_image: "%s"
images: ["%s"]
cover:
    image: "%s"
---

%s

**Fuente:** [%s](%s)

---
*Analizado por Micomicona AI*
`, 
		escapeYAML(item.Title),
		pubDate.In(time.Local).Format("2006-01-02 15:04"),
		sourceName,
		tagsStr,
		escapeYAML(summary),
		imageURL,
		imageURL,
		imageURL,
		cleanDescription,
		sourceName,
		item.Link,
	)

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Printf("Error escribiendo %s: %v", fileName, err)
	}
}
