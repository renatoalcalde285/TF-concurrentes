package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
)

type Rating struct {
	UserID  string
	MovieID string
	Rating  float64
}

type ClientData struct {
	TargetUserID string
	Data         []Rating
}

type recommendationPair struct {
	MovieID string
	Rating  float64
}

var addrs []string
var hostIP string

// Servicios
const (
	portHP = 9002
)

func main() {
	hostIP = descubrirIP()
	fmt.Printf("Mi IP es %s\n", hostIP)

	addrs = []string{
		"172.30.0.5",
	}

	servicioHP()
}

func descubrirIP() string {
	var dirIP string = "127.0.0.1"
	interfaces, _ := net.Interfaces()
	for _, valInterface := range interfaces {
		if strings.HasPrefix(valInterface.Name, "eth0") {
			direcciones, _ := valInterface.Addrs()
			for _, valDireccion := range direcciones {
				switch d := valDireccion.(type) {
				case *net.IPNet:
					if d.IP.To4() != nil {
						dirIP = d.IP.String()
					}
				}
			}
		}
	}
	return dirIP
}

func servicioHP() {
	localDir := fmt.Sprintf("%s:%d", hostIP, portHP)

	ln, err := net.Listen("tcp", localDir)
	if err != nil {
		fmt.Printf("Error al iniciar el servicio HP: %v\n", err)
		return
	}
	defer ln.Close()

	fmt.Printf("Servicio HP escuchando en %s\n", localDir)
	for {
		con, err := ln.Accept()
		if err != nil {
			fmt.Printf("Error aceptando conexión: %v\n", err)
			continue
		}
		go handlerHP(con)
	}
}

func handlerHP(con net.Conn) {
	defer func() {
		fmt.Println("Conexión cerrada con el cliente.")
		con.Close()
	}()

	scanner := bufio.NewScanner(con)
	clientData := ClientData{
		TargetUserID: "",
		Data:         make([]Rating, 0),
	}

	fmt.Println("Recibiendo datos del cliente...")
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "UserID:") {
			clientData.TargetUserID = strings.TrimSpace(line[len("UserID:"):])
		} else {
			fields := strings.Split(line, ",")
			if len(fields) == 3 {
				userID := strings.TrimSpace(fields[0])
				movieID := strings.TrimSpace(fields[1])
				rating, err3 := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
				if err3 == nil {
					clientData.Data = append(clientData.Data, Rating{
						UserID:  userID,
						MovieID: movieID,
						Rating:  rating,
					})
				} else {
					fmt.Println("Error procesando línea:", line)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Error leyendo datos:", err)
	}

	// Mostrar la data recibida
	fmt.Println("\nData recibida del cliente:")
	fmt.Printf("UserID objetivo: %s\n", clientData.TargetUserID)

	// Crear la matriz usuario-item
	userItemMatrix := createUserItemMatrix(clientData)

	// Realizar la factorización de la matriz
	fmt.Println("\nRealizando la factorización de la matriz con SGD...")
	numFactors := 3
	learningRate := 0.01
	numIterations := 10

	userFactors, itemFactors := matrixFactorizationWithSGD(userItemMatrix, numFactors, learningRate, numIterations)

	// Calcular y obtener las recomendaciones
	recommendations := calculateRecommendations(clientData.TargetUserID, userItemMatrix, userFactors, itemFactors)

	var sortedRecommendations []recommendationPair
	for movieID, score := range recommendations {
		sortedRecommendations = append(sortedRecommendations, recommendationPair{
			MovieID: movieID,
			Rating:  score,
		})
	}

	// Ordenar las recomendaciones de mayor a menor rating
	sort.Slice(sortedRecommendations, func(i, j int) bool {
		return sortedRecommendations[i].Rating > sortedRecommendations[j].Rating
	})

	// Mostrar las primeras 5 recomendaciones ordenadas
	fmt.Println("\nPrimeras 5 recomendaciones ordenadas:")
	count := 0
	for _, rec := range sortedRecommendations {
		fmt.Printf("MovieID: %s, Predicted Rating: %.2f\n", rec.MovieID, rec.Rating)
		count++
		if count >= 5 {
			break
		}
	}

	// Enviar el Top 5 al servidor
	enviarTop5(con, sortedRecommendations[:5])

	// Mostrar el largo de las recomendaciones
	fmt.Printf("\nCantidad total de recomendaciones generadas: %d\n", len(recommendations))
}

// Función para enviar el Top 5 de recomendaciones al servidor
func enviarTop5(conn net.Conn, top5 []recommendationPair) {
	for _, rec := range top5 {
		// Enviar cada recomendación (MovieID y Rating) al servidor
		fmt.Fprintf(conn, "%s,%.2f\n", rec.MovieID, rec.Rating)
	}
}

func createUserItemMatrix(clientData ClientData) map[string]map[string]float64 {
	matrix := make(map[string]map[string]float64)
	for _, rating := range clientData.Data {
		if _, exists := matrix[rating.UserID]; !exists {
			matrix[rating.UserID] = make(map[string]float64)
		}
		matrix[rating.UserID][rating.MovieID] = rating.Rating
	}
	return matrix
}

func matrixFactorizationWithSGD(matrix map[string]map[string]float64, numFactors int, learningRate float64, numIterations int) (map[string][]float64, map[string][]float64) {
	// Inicializar factores aleatorios
	userFactors := make(map[string][]float64)
	itemFactors := make(map[string][]float64)

	// Inicialización de factores aleatorios para usuarios
	for userID := range matrix {
		factors := make([]float64, numFactors)
		for i := 0; i < numFactors; i++ {
			factors[i] = rand.Float64()
		}
		userFactors[userID] = factors
	}

	// Inicialización de factores aleatorios para ítems (películas)
	itemSet := make(map[string]bool)
	for _, movies := range matrix {
		for movieID := range movies {
			itemSet[movieID] = true
		}
	}

	for movieID := range itemSet {
		factors := make([]float64, numFactors)
		for i := 0; i < numFactors; i++ {
			factors[i] = rand.Float64()
		}
		itemFactors[movieID] = factors
	}

	// Entrenamiento con SGD
	for iter := 0; iter < numIterations; iter++ {
		for userID, movies := range matrix {
			for movieID, actualRating := range movies {
				// Predicción de calificación
				predictedRating := predictRating(userFactors[userID], itemFactors[movieID])

				// Error entre la calificación real y la predicha
				error := actualRating - predictedRating

				// Actualización de factores
				for k := 0; k < numFactors; k++ {
					// Ajuste para factores del usuario
					userFactors[userID][k] += learningRate * error * itemFactors[movieID][k]
					// Ajuste para factores del ítem
					itemFactors[movieID][k] += learningRate * error * userFactors[userID][k]
				}
			}
		}
		fmt.Printf("Iteración %d completada\n", iter+1)
	}

	return userFactors, itemFactors
}

func predictRating(userFactors, itemFactors []float64) float64 {
	var predictedRating float64
	for i := 0; i < len(userFactors); i++ {
		predictedRating += userFactors[i] * itemFactors[i]
	}
	return predictedRating
}

func calculateRecommendations(targetUserID string, userItemMatrix map[string]map[string]float64, userFactors map[string][]float64, itemFactors map[string][]float64) map[string]float64 {
	recommendations := make(map[string]float64)

	// Generar recomendaciones para el usuario objetivo
	for movieID := range itemFactors {
		if _, rated := userItemMatrix[targetUserID][movieID]; !rated {
			// Calcular predicción
			predictedRating := predictRating(userFactors[targetUserID], itemFactors[movieID])
			recommendations[movieID] = predictedRating
		}
	}

	return recommendations
}
