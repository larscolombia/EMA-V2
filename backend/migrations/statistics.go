package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// RecordTestCompletion registra un test completado en el historial
func RecordTestCompletion(userID int, categoryID *int, testName string, scoreObtained, maxScore int) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}

	log.Printf("[STATS] Recording test: userID=%d categoryID=%v testName=%s score=%d/%d",
		userID, categoryID, testName, scoreObtained, maxScore)

	_, err := db.Exec(
		"INSERT INTO test_history (user_id, category_id, test_name, score_obtained, max_score) VALUES (?, ?, ?, ?, ?)",
		userID, categoryID, testName, scoreObtained, maxScore,
	)
	if err != nil {
		log.Printf("[STATS] ❌ Error recording test: userID=%d testName=%s error=%v", userID, testName, err)
		return fmt.Errorf("database error: %v", err)
	}
	log.Printf("[STATS] ✅ Test recorded successfully: userID=%d testName=%s score=%d/%d", userID, testName, scoreObtained, maxScore)
	return nil
}

// GetTestProgress obtiene estadísticas generales de progreso del usuario
func GetTestProgress(userID int, limit int) (map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("db is not initialized")
	}

	// Obtener resumen general (promedio de TODOS los tests)
	summaryQuery := `
		SELECT 
			COUNT(*) as total_tests,
			IFNULL(SUM(score_obtained), 0) as total_score_obtained,
			IFNULL(SUM(max_score), 0) as total_max_score
		FROM test_history
		WHERE user_id = ?`

	var totalTests, totalScoreObtained, totalMaxScore int
	err := db.QueryRow(summaryQuery, userID).Scan(&totalTests, &totalScoreObtained, &totalMaxScore)
	if err != nil {
		return nil, err
	}

	// Calcular promedio general
	averagePercentage := 0.0
	if totalMaxScore > 0 {
		averagePercentage = (float64(totalScoreObtained) / float64(totalMaxScore)) * 100
	}

	// Obtener últimos tests para detalle
	detailQuery := `
		SELECT th.id, th.test_name, th.score_obtained, th.max_score, th.created_at,
		       IFNULL(mc.name, '') as category_name
		FROM test_history th
		LEFT JOIN medical_categories mc ON th.category_id = mc.id
		WHERE th.user_id = ?
		ORDER BY th.created_at DESC
		LIMIT ?`

	rows, err := db.Query(detailQuery, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recentTests []map[string]interface{}
	for rows.Next() {
		var id, scoreObtained, maxScore int
		var testName, categoryName string
		var createdAt time.Time

		if err := rows.Scan(&id, &testName, &scoreObtained, &maxScore, &createdAt, &categoryName); err != nil {
			continue
		}

		percentage := 0.0
		if maxScore > 0 {
			percentage = (float64(scoreObtained) / float64(maxScore)) * 100
		}

		recentTests = append(recentTests, map[string]interface{}{
			"test_id":        id,
			"test_name":      testName,
			"score_obtained": scoreObtained,
			"max_score":      maxScore,
			"percentage":     percentage,
			"category_name":  categoryName,
			"created_at":     createdAt.Format(time.RFC3339),
		})
	}

	// Retornar resumen + últimos tests
	return map[string]interface{}{
		"summary": map[string]interface{}{
			"total_tests":        totalTests,
			"total_score":        totalScoreObtained,
			"total_max_score":    totalMaxScore,
			"average_percentage": averagePercentage,
		},
		"recent_tests": recentTests,
	}, nil
}

// GetMonthlyScores obtiene los puntajes agrupados por mes (últimos 6 meses)
func GetMonthlyScores(userID int) ([]map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("db is not initialized")
	}

	query := `
		SELECT 
			DATE_FORMAT(created_at, '%Y-%m') as mes,
			SUM(score_obtained) as puntos,
			COUNT(*) as tests_count
		FROM test_history
		WHERE user_id = ? 
		  AND created_at >= DATE_SUB(NOW(), INTERVAL 6 MONTH)
		GROUP BY DATE_FORMAT(created_at, '%Y-%m')
		ORDER BY mes DESC
		LIMIT 6`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var mes string
		var puntos, testsCount int

		if err := rows.Scan(&mes, &puntos, &testsCount); err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"mes":         mes,
			"puntos":      puntos,
			"tests_count": testsCount,
		})
	}

	return results, nil
}

// GetMostStudiedCategory obtiene la categoría más estudiada del usuario
func GetMostStudiedCategory(userID int) (map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("db is not initialized")
	}

	query := `
		SELECT mc.id, mc.name, COUNT(*) as study_count
		FROM test_history th
		JOIN medical_categories mc ON th.category_id = mc.id
		WHERE th.user_id = ?
		GROUP BY mc.id, mc.name
		ORDER BY study_count DESC
		LIMIT 1`

	row := db.QueryRow(query, userID)

	var categoryID, studyCount int
	var categoryName string

	err := row.Scan(&categoryID, &categoryName, &studyCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No hay categoría estudiada
		}
		return nil, err
	}

	return map[string]interface{}{
		"category_id":   categoryID,
		"category_name": categoryName,
		"study_count":   studyCount,
	}, nil
}
