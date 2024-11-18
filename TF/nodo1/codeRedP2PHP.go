package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Rating struct {
	UserID  int
	MovieID int
	Rating  float64
}

type ClientData struct {
	TargetUserID string
	Data         []Rating
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
		"172.20.0.5",
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
	// Modo escucha
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
		fmt.Println("Conexión cerrada con el servidor.")
		con.Close()
	}()

	scanner := bufio.NewScanner(con)
	clientData := ClientData{
		TargetUserID: "",
		Data:         make([]Rating, 0),
	}

	// Recibir y procesar data del servidor
	fmt.Println("Recibiendo datos del servidor...")
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
				userID, err1 := strconv.Atoi(strings.TrimSpace(fields[0]))
				movieID, err2 := strconv.Atoi(strings.TrimSpace(fields[1]))
				rating, err3 := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
				if err1 == nil && err2 == nil && err3 == nil {
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
		fmt.Println("Error leyendo del servidor:", err)
	}

	fmt.Println("\nData recibida del servidor:")
	fmt.Printf("UserID objetivo: %s\n", clientData.TargetUserID)

	// Mostrar los primeros elementos del array recibido
	fmt.Printf("Cantidad total de elementos: %d\n", len(clientData.Data))
	fmt.Println("Primeros 5 elementos:")
	for i, rating := range clientData.Data {
		if i >= 5 {
			break
		}
		fmt.Printf("UserID: %d, MovieID: %d, Rating: %.2f\n", rating.UserID, rating.MovieID, rating.Rating)
	}
}
