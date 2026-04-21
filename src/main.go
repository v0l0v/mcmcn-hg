package main

import (
	"bufio"
	"fmt"
	"log"
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

func main() {
	fmt.Println("Iniciando Micomicona Fetcher (Universal Images + AI)...")
	log.Println("IA desactivada")

	// IA desactivada: no se carga GEMINI_API_KEY


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

// Función getAPIKey eliminada – IA desactivada

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
    // Gemini desactivado: devolvemos valores vacíos para evitar llamadas externas.
    // Mantener la firma para compatibilidad con el resto del código.
    return "", false
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
	// Si el archivo ya existe, lo sobrescribimos para actualizar fecha y contenido

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
featured_image: "%s"
images: ["%s"]
cover:
    image: "%s"
---

%s

**Fuente:** [%s](%s)

---
*Analizado por Micomicona AI*`,
		escapeYAML(item.Title),
		pubDate.In(time.Local).Format("2006-01-02 15:04"),
		sourceName,
		tagsStr,
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
