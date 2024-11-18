package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Rating struct {
	UserID  string
	MovieID string
	Rating  float64
}

type ClientData struct {
	UserID string
	Data   []Rating
}

type Recommendation struct {
	MovieID string
	Rating  float64
}

var (
	hostIP      string
	addrs       []string
	dataset     map[string][]Rating
	numNodes    = 3
	topGlobal   []Recommendation
	muTopGlobal sync.Mutex
	wg          sync.WaitGroup
)

// Servicios
const (
	portHP = 9002
)

func main() {

	hostIP = descubrirIP()
	fmt.Printf("IP del Servidor: %s\n", hostIP)

	addrs = []string{
		"172.30.0.2", "172.30.0.3", "172.30.0.4"}

	// Cargar el dataset
	var err error
	dataset, err = loadDataset("/app/dataset.csv", 2000000)
	if err != nil {
		log.Fatalf("Error cargando el dataset: %v", err)
	}
	fmt.Println("Dataset cargado correctamente.")

	targetUserID := "589967"

	targetUserRatings, exists := dataset[targetUserID]
	if !exists {
		log.Fatalf("El UserID %s no existe en el dataset", targetUserID)
	}

	// Dividir el dataset
	clientData, targetCount := splitDataset(dataset, targetUserRatings, numNodes)

	fmt.Printf("Se encontraron %d elementos para el usuario objetivo %s\n", targetCount, targetUserID)

	// Mostrar los primeros elementos de cada subconjunto
	for i, data := range clientData {
		fmt.Printf("\nSubconjunto %d tiene %d elementos.\n", i+1, len(data.Data))
		fmt.Printf("Primeros 5 valores del cliente %d:\n", i+1)
		for j, rating := range data.Data[:5] {
			fmt.Printf("  %d. UserID: %s, MovieID: %s, Rating: %.2f\n", j+1, rating.UserID, rating.MovieID, rating.Rating)
		}
	}

	time.Sleep(20 * time.Second)

	iniciarServidor(clientData)
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

// Cargar dataset
func loadDataset(filename string, limit int) (map[string][]Rating, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	userRatings := make(map[string][]Rating)
	if limit <= 0 || limit > len(records)-1 {
		limit = len(records) - 1
	}

	for _, record := range records[1 : limit+1] {
		movieID := record[0]
		userID := record[1]
		rating, _ := strconv.ParseFloat(record[2], 64)

		userRatings[userID] = append(userRatings[userID], Rating{
			UserID:  userID,
			MovieID: movieID,
			Rating:  rating,
		})
	}
	return userRatings, nil
}

// Dividir dataset
func splitDataset(userRatings map[string][]Rating, targetUserRatings []Rating, numClients int) ([]ClientData, int) {
	clientData := make([]ClientData, numClients)
	targetCount := len(targetUserRatings)

	for i := 0; i < numClients; i++ {
		clientData[i].UserID = targetUserRatings[0].UserID
		for _, rating := range targetUserRatings {
			clientData[i].Data = append(clientData[i].Data, rating)
		}
	}

	delete(userRatings, targetUserRatings[0].UserID)

	i := 0
	for _, ratings := range userRatings {
		for _, rating := range ratings {
			clientData[i%numClients].Data = append(clientData[i%numClients].Data, rating)
			i++
		}
	}

	return clientData, targetCount
}

// Enviar data a clientes
func enviarDatos(clientData []ClientData) {
	rand.Seed(time.Now().UnixNano())
	for i, clienteIP := range addrs {
		remoteDir := fmt.Sprintf("%s:%d", clienteIP, portHP)
		conn, err := net.Dial("tcp", remoteDir)
		if err != nil {
			fmt.Printf("Error conectando con %s: %v\n", clienteIP, err)
			continue
		}
		defer conn.Close()

		// Enviar datos al cliente
		fmt.Fprintf(conn, "UserID: %s\n", clientData[i].UserID)
		for _, rating := range clientData[i].Data {
			fmt.Fprintf(conn, "%s,%s,%.2f\n", rating.UserID, rating.MovieID, rating.Rating)
		}
		fmt.Printf("Datos enviados a %s\n", clienteIP)
	}
}

// Iniciar servidor
func iniciarServidor(clientData []ClientData) {
	localDir := fmt.Sprintf("%s:%d", hostIP, portHP)

	ln, err := net.Listen("tcp", localDir)
	if err != nil {
		fmt.Printf("Error iniciando el servidor: %v\n", err)
		return
	}
	defer ln.Close()

	fmt.Println("Servidor escuchando en", localDir)

	time.Sleep(20 * time.Second)

	enviarDatos(clientData)

	for i := 0; i < numNodes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			con, err := ln.Accept()
			if err != nil {
				fmt.Printf("Error aceptando conexión: %v\n", err)
				return
			}
			manejarConexion(con)
		}()
	}

	wg.Wait()

	// Calcular el Top 3 final
	calcularTop3Final()
}

// Manejar conexiones de los clientes
func manejarConexion(con net.Conn) {
	defer con.Close()

	reader := bufio.NewReader(con)
	tempResults := []Recommendation{}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Printf("Error leyendo datos: %v\n", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "FIN_TOP5" {
			fmt.Println("Fin del Top 5 recibido.")
			break
		}

		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			fmt.Printf("Datos no válidos: %s\n", line)
			continue
		}

		movieID := parts[0]
		rating, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			fmt.Printf("Error al parsear el rating: %v\n", err)
			continue
		}

		tempResults = append(tempResults, Recommendation{
			MovieID: movieID,
			Rating:  rating,
		})
	}

	// Actualizar el Top Global
	actualizarTopGlobal(tempResults)
}

func actualizarTopGlobal(tempResults []Recommendation) {
	muTopGlobal.Lock()
	defer muTopGlobal.Unlock()

	// Mapa para calcular promedios
	ratingMap := make(map[string][]float64)

	// Incluir las películas ya existentes
	for _, rec := range topGlobal {
		ratingMap[rec.MovieID] = append(ratingMap[rec.MovieID], rec.Rating)
	}

	// Agregar las nuevas recomendaciones
	for _, rec := range tempResults {
		ratingMap[rec.MovieID] = append(ratingMap[rec.MovieID], rec.Rating)
	}

	// Construir el nuevo Top Global
	newTopGlobal := []Recommendation{}
	for movieID, ratings := range ratingMap {
		// Si el rating tiene más de una entrada, promediamos
		newTopGlobal = append(newTopGlobal, Recommendation{
			MovieID: movieID,
			Rating:  promedio(ratings),
		})
	}

	// Ordenar por rating descendente
	sort.Slice(newTopGlobal, func(i, j int) bool {
		return newTopGlobal[i].Rating > newTopGlobal[j].Rating
	})

	// Limitar a los Top 15
	if len(newTopGlobal) > 15 {
		newTopGlobal = newTopGlobal[:15]
	}

	topGlobal = newTopGlobal
	fmt.Println("Top Global actualizado:", topGlobal)
}

func calcularTop3Final() {
	muTopGlobal.Lock()
	defer muTopGlobal.Unlock()

	// Si hay más de 3, limitar a 3
	if len(topGlobal) > 3 {
		// Tomamos solo las primeras 3 recomendaciones
		topGlobal = topGlobal[:3]

		// Promediamos los ratings de las películas repetidas
		seenMovies := make(map[string][]float64)
		for _, rec := range topGlobal {
			seenMovies[rec.MovieID] = append(seenMovies[rec.MovieID], rec.Rating)
		}

		// Reemplazar los ratings por el promedio si hay duplicados
		for i, rec := range topGlobal {
			if len(seenMovies[rec.MovieID]) > 1 {
				topGlobal[i].Rating = promedio(seenMovies[rec.MovieID])
			}
		}
	}

	// Mostrar el Top 3 con el formato solicitado
	fmt.Println("Top 3 final:")
	for _, rec := range topGlobal {
		fmt.Printf("MovieID: %s, Predicted Rating: %.2f\n", rec.MovieID, rec.Rating)
	}
}

func promedio(nums []float64) float64 {
	sum := 0.0
	for _, num := range nums {
		sum += num
	}
	return sum / float64(len(nums))
}
