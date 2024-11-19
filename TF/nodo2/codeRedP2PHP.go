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
				rating, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
				if err == nil {
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
	fmt.Println("\nData recibida del cliente:")
	fmt.Printf("UserID objetivo: %s\n", clientData.TargetUserID)

	userItemMatrix := createUserItemMatrix(clientData)

	// Realizar la factorización de la matriz
	fmt.Println("\nRealizando la factorización de la matriz con SGD...")
	numFactors := 3
	learningRate := 0.01
	numIterations := 10
	userFactors, itemFactors := matrixFactorizationWithSGD(userItemMatrix, numFactors, learningRate, numIterations)
	recommendations := calculateRecommendations(clientData.TargetUserID, userItemMatrix, userFactors, itemFactors)

	var sortedRecommendations []recommendationPair
	for movieID, score := range recommendations {
		sortedRecommendations = append(sortedRecommendations, recommendationPair{
			MovieID: movieID,
			Rating:  score,
		})
	}
	sort.Slice(sortedRecommendations, func(i, j int) bool {
		return sortedRecommendations[i].Rating > sortedRecommendations[j].Rating
	})
	fmt.Println("\nPrimeras 5 recomendaciones ordenadas:")
	count := 0
	for _, rec := range sortedRecommendations {
		fmt.Printf("MovieID: %s, Predicted Rating: %.2f\n", rec.MovieID, rec.Rating)
		count++
		if count >= 5 {
			break
		}
	}
	enviarRecomendacionesAlServidor(sortedRecommendations[:5])
	fmt.Printf("\nCantidad total de recomendaciones generadas: %d\n", len(recommendations))
}

func enviarRecomendacionesAlServidor(recommendations []recommendationPair) {
	serverAddr := fmt.Sprintf("%s:%d", addrs[0], portHP)
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("Error conectando con el servidor: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Enviando las recomendaciones al servidor...")
	for _, rec := range recommendations {
		fmt.Fprintf(conn, "%s,%.2f\n", rec.MovieID, rec.Rating)
	}
	fmt.Fprintln(conn, "FIN_TOP5")
	fmt.Println("Recomendaciones enviadas al servidor.")
}

func enviarTop5(conn net.Conn, top5 []recommendationPair) {
	for _, rec := range top5 {
		fmt.Fprintf(conn, "%s,%.2f\n", rec.MovieID, rec.Rating)
		fmt.Fprintln(conn, "FIN_TOP5")
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

	for userID := range matrix {
		factors := make([]float64, numFactors)
		for i := 0; i < numFactors; i++ {
			factors[i] = rand.Float64()
		}
		userFactors[userID] = factors
	}

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

	for iter := 0; iter < numIterations; iter++ {
		for userID, movies := range matrix {
			for movieID, actualRating := range movies {
				// Predicción de calificación
				predictedRating := predictRating(userFactors[userID], itemFactors[movieID])
				error := actualRating - predictedRating
				for k := 0; k < numFactors; k++ {
					userFactors[userID][k] += learningRate * error * itemFactors[movieID][k]
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
			predictedRating := predictRating(userFactors[targetUserID], itemFactors[movieID])
			recommendations[movieID] = predictedRating
		}
	}

	return recommendations
}
