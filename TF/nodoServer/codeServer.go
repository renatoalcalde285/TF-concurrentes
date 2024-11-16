package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

var hostIP string
var addrs []string

// Servicios
const (
	portHP = 9002 //Servicio hp
)

func main() {
	// Obtener la IP del servidor
	hostIP = descubrirIP()
	fmt.Printf("IP del Servidor: %s\n", hostIP)

	// Inicializar la lista de nodos clientes
	addrs = []string{
		"172.20.0.2", "172.20.0.3", "172.20.0.4"}

	// Iniciar el servicio
	iniciarServidor()
}

// Función para descubrir la IP del nodo
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

// Función para iniciar el servidor
func iniciarServidor() {
	// Dirección local en la que escucha el servidor
	localDir := fmt.Sprintf("%s:%d", hostIP, portHP)

	ln, err := net.Listen("tcp", localDir)
	if err != nil {
		fmt.Printf("Error iniciando el servidor: %v\n", err)
		return
	}
	defer ln.Close()

	fmt.Println("Servidor escuchando en", localDir)

	time.Sleep(10 * time.Second)

	enviarMsg()

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

// Función para enviar un número aleatorio a cada cliente
func enviarMsg() {
	// Semilla para números aleatorios
	rand.Seed(time.Now().UnixNano())

	for _, clienteIP := range addrs {
		// Generar número aleatorio entre 1 y 10
		numero := rand.Intn(10) + 1

		// Dirección del cliente
		remoteDir := fmt.Sprintf("%s:%d", clienteIP, portHP)

		// Conectar al cliente
		conn, err := net.Dial("tcp", remoteDir)
		if err != nil {
			fmt.Printf("Error conectando con %s: %v\n", clienteIP, err)
			continue
		}
		defer conn.Close()

		// Enviar el número
		fmt.Fprintf(conn, "%d\n", numero)
		fmt.Printf("Número %d enviado a %s\n", numero, clienteIP)
	}
}
