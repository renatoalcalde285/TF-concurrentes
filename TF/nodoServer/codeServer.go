package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
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

var hostIP string
var addrs []string
var dataset map[string][]Rating
var numNodes = 3

// Servicios
const (
	portHP = 9002
)

func main() {

	hostIP = descubrirIP()
	fmt.Printf("IP del Servidor: %s\n", hostIP)

	addrs = []string{
		"172.20.0.2", "172.20.0.3", "172.20.0.4"}

	// Cargar el dataset
	var err error
	dataset, err = loadDataset("/app/dataset2M.csv", 100)
	if err != nil {
		log.Fatalf("Error cargando el dataset: %v", err)
	}
	fmt.Println("Dataset cargado correctamente.")

	targetUserID := "30878"

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
	// Dirección local en la que escucha el servidor
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

	for {
		con, err := ln.Accept()
		if err != nil {
			fmt.Printf("Error aceptando conexión: %v\n", err)
			continue
		}
		go manejarConexion(con)
	}
}

// Manejar conexiones de los clientes
func manejarConexion(con net.Conn) {
	defer con.Close()

	// Leer datos enviados por el cliente
	mensaje, err := bufio.NewReader(con).ReadString('\n')
	if err != nil {
		fmt.Printf("Error leyendo datos: %v\n", err)
		return
	}

	fmt.Println(mensaje)

	// Procesar y responder al cliente
	/*res := con.RemoteAddr().String()
	fmt.Printf("Mensaje recibido de %s: %s\n", res, strings.TrimSpace(mensaje))
	fmt.Fprintln(con, "Respuesta del servidor: Recibido")*/
}
