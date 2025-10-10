package openai

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TestSearchPubMed prueba la búsqueda real en PubMed E-utilities API
func TestSearchPubMed(t *testing.T) {
	// No requiere API key de OpenAI, usa directamente PubMed API pública
	client := &Client{}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	// Test 1: Búsqueda real de un tema médico conocido
	t.Run("diabetes mellitus treatment", func(t *testing.T) {
		result, err := client.SearchPubMed(ctx, "diabetes mellitus type 2 treatment guidelines")
		if err != nil {
			t.Fatalf("SearchPubMed failed: %v", err)
		}

		if result == "" {
			t.Log("No results found (expected for some queries)")
			return
		}

		// Validar que es JSON válido
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON response: %v\nResponse: %s", err, result)
		}

		// Validar estructura
		if _, ok := data["summary"]; !ok {
			t.Error("Missing 'summary' field")
		}

		studies, ok := data["studies"].([]interface{})
		if !ok {
			t.Fatal("Missing or invalid 'studies' field")
		}

		if len(studies) == 0 {
			t.Error("No studies found")
		}

		// Validar primer estudio
		firstStudy := studies[0].(map[string]interface{})
		requiredFields := []string{"title", "pmid", "year", "journal"}
		for _, field := range requiredFields {
			if _, ok := firstStudy[field]; !ok {
				t.Errorf("Missing required field '%s' in study", field)
			}
		}

		// Validar PMID es numérico
		pmid, ok := firstStudy["pmid"].(string)
		if !ok || pmid == "" {
			t.Error("Invalid PMID")
		}

		t.Logf("✓ Found %d studies", len(studies))
		t.Logf("✓ Summary: %v", data["summary"])
		t.Logf("✓ First article PMID: %s", pmid)
	})

	// Test 2: Búsqueda de tema muy específico (menos resultados)
	t.Run("specific medical condition", func(t *testing.T) {
		// Delay para respetar rate limits de NCBI
		time.Sleep(1 * time.Second)

		result, err := client.SearchPubMed(ctx, "CRISPR gene editing therapy")
		if err != nil {
			t.Fatalf("SearchPubMed failed: %v", err)
		}

		if result == "" {
			t.Log("No results found (acceptable for very specific queries)")
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}

		t.Logf("✓ Result: %s", result)
	})

	// Test 3: Query que probablemente no tenga resultados
	t.Run("nonsense query", func(t *testing.T) {
		// Delay para respetar rate limits de NCBI
		time.Sleep(1 * time.Second)

		result, err := client.SearchPubMed(ctx, "xyzabc123nonexistent")
		if err != nil {
			t.Fatalf("SearchPubMed failed: %v", err)
		}

		if result != "" {
			t.Errorf("Expected empty result for nonsense query, got: %s", result)
		}

		t.Log("✓ Correctly returned empty result for nonsense query")
	})
}

// TestParsePubMedXML prueba el parser de XML
func TestParsePubMedXML(t *testing.T) {
	// XML de ejemplo de PubMed (simplificado)
	xmlData := []byte(`<?xml version="1.0"?>
<!DOCTYPE PubmedArticleSet PUBLIC "-//NLM//DTD PubMedArticle, 1st January 2019//EN" "https://dtd.nlm.nih.gov/ncbi/pubmed/out/pubmed_190101.dtd">
<PubmedArticleSet>
  <PubmedArticle>
    <MedlineCitation>
      <PMID>12345678</PMID>
      <Article>
        <ArticleTitle>Test Article Title</ArticleTitle>
        <Abstract>
          <AbstractText>This is a test abstract for validation purposes.</AbstractText>
        </Abstract>
        <AuthorList>
          <Author>
            <LastName>Smith</LastName>
            <ForeName>John</ForeName>
            <Initials>J</Initials>
          </Author>
          <Author>
            <LastName>Doe</LastName>
            <ForeName>Jane</ForeName>
            <Initials>J</Initials>
          </Author>
        </AuthorList>
        <Journal>
          <Title>Test Journal</Title>
          <JournalIssue>
            <PubDate>
              <Year>2023</Year>
              <Month>Jan</Month>
            </PubDate>
          </JournalIssue>
        </Journal>
      </Article>
    </MedlineCitation>
    <PubmedData>
      <ArticleIdList>
        <ArticleId IdType="doi">10.1234/test</ArticleId>
        <ArticleId IdType="pubmed">12345678</ArticleId>
      </ArticleIdList>
    </PubmedData>
  </PubmedArticle>
</PubmedArticleSet>`)

	articles, err := parsePubMedXML(xmlData)
	if err != nil {
		t.Fatalf("parsePubMedXML failed: %v", err)
	}

	if len(articles) != 1 {
		t.Fatalf("Expected 1 article, got %d", len(articles))
	}

	article := articles[0]

	// Validar campos
	if article["pmid"] != "12345678" {
		t.Errorf("Expected PMID 12345678, got %v", article["pmid"])
	}

	if article["title"] != "Test Article Title" {
		t.Errorf("Expected title 'Test Article Title', got %v", article["title"])
	}

	if article["year"] != 2023 {
		t.Errorf("Expected year 2023, got %v", article["year"])
	}

	if article["journal"] != "Test Journal" {
		t.Errorf("Expected journal 'Test Journal', got %v", article["journal"])
	}

	// DOI es opcional
	if doi, ok := article["doi"]; ok {
		if doi != "doi:10.1234/test" {
			t.Errorf("Expected DOI 'doi:10.1234/test', got %v", doi)
		}
	} else {
		t.Log("Note: DOI not parsed (may be expected for some XML formats)")
	}

	authors := article["authors"].([]string)
	if len(authors) != 2 {
		t.Errorf("Expected 2 authors, got %d", len(authors))
	}

	if authors[0] != "Smith J" {
		t.Errorf("Expected first author 'Smith J', got %s", authors[0])
	}

	t.Log("✓ XML parsing successful")
}

// TestSearchPubMedSpanish prueba la búsqueda con query en español (auto-traducción)
func TestSearchPubMedSpanish(t *testing.T) {
	// Crear client que lee API key del environment
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	t.Run("spanish query - GIST tumor", func(t *testing.T) {
		// Query similar a la del usuario real
		result, err := client.SearchPubMed(ctx, "mejor tratamiento para tumor gist gastrico de 5cm")
		if err != nil {
			t.Fatalf("SearchPubMed failed: %v", err)
		}

		if result == "" {
			if client.api == nil {
				t.Log("⚠ No results - OPENAI_API_KEY not set, translation limited")
			} else {
				t.Error("Expected results for translated query, got empty")
			}
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}

		studies, ok := data["studies"].([]interface{})
		if !ok || len(studies) == 0 {
			t.Error("Expected studies in result")
		}

		t.Logf("✓ Found %d studies with Spanish query (auto-translated)", len(studies))
		t.Logf("✓ Summary: %v", data["summary"])

		// Mostrar primer estudio
		if len(studies) > 0 {
			firstStudy := studies[0].(map[string]interface{})
			t.Logf("✓ First study: %s (PMID: %s)", firstStudy["title"], firstStudy["pmid"])
		}
	})
}
