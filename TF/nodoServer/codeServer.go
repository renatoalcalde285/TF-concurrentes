// codeServer.go

package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
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
    MovieID string  `json:"MovieID"`
    Rating  float64 `json:"Rating"`
}

var (
    hostIP      string
    addrs       []string
    dataset     map[string][]Rating
    numNodes    = 5
    topGlobal   []Recommendation
    muTopGlobal sync.Mutex
    wg          sync.WaitGroup
)

const (
    portHP = 9002
)

func main() {
    hostIP = descubrirIP()
    fmt.Printf("IP del Servidor: %s\n", hostIP)

    addrs = []string{
        "172.30.0.2", "172.30.0.3", "172.30.0.4", "172.30.0.6", "172.30.0.7",
    }

    // Cargar el dataset
    var err error
    dataset, err = loadDataset("/app/dataset2M.csv", 2000000)
    if err != nil {
        log.Fatalf("Error cargando el dataset: %v", err)
    }
    fmt.Println("Dataset cargado correctamente.")

    // Iniciar el servidor HTTP
    http.HandleFunc("/recommend", recommendationHandler)
    fmt.Println("Iniciando el servidor HTTP en el puerto 8080...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func recommendationHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    userID := r.URL.Query().Get("userId")
    if userID == "" {
        http.Error(w, "userId parameter is required", http.StatusBadRequest)
        return
    }

    recommendations, err := generateRecommendations(userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string][]Recommendation{
        "recommendations": recommendations,
    })
}

func generateRecommendations(userID string) ([]Recommendation, error) {
    // Copy the dataset to avoid modifying the original
    userRatingsCopy := make(map[string][]Rating)
    for k, v := range dataset {
        userRatingsCopy[k] = v
    }

    clientData, targetCount := splitDataset(userRatingsCopy, userID, numNodes)
    if targetCount == 0 {
        return nil, fmt.Errorf("No data found for user %s", userID)
    }

    recommendations, err := iniciarServidor(clientData)
    if err != nil {
        return nil, err
    }

    return recommendations, nil
}

func iniciarServidor(clientData []ClientData) ([]Recommendation, error) {
    localDir := fmt.Sprintf("%s:%d", hostIP, portHP)

    ln, err := net.Listen("tcp", localDir)
    if err != nil {
        fmt.Printf("Error iniciando el servidor: %v\n", err)
        return nil, err
    }
    defer ln.Close()

    fmt.Println("Servidor escuchando en", localDir)

    // Enviar datos a los nodos
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

    // Calcular el top 3 final
    finalTop3 := calcularTop3Final()

    return finalTop3, nil
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

func splitDataset(userRatings map[string][]Rating, targetUserID string, numClients int) ([]ClientData, int) {
    targetUserRatings, exists := userRatings[targetUserID]
    if !exists {
        log.Printf("UserID %s does not exist in the dataset", targetUserID)
        return nil, 0
    }

    clientData := make([]ClientData, numClients)
    targetCount := len(targetUserRatings)

    for i := 0; i < numClients; i++ {
        clientData[i].UserID = targetUserID
        clientData[i].Data = append(clientData[i].Data, targetUserRatings...)
    }

    delete(userRatings, targetUserID)

    i := 0
    for _, ratings := range userRatings {
        for _, rating := range ratings {
            clientData[i%numClients].Data = append(clientData[i%numClients].Data, rating)
            i++
        }
    }

    return clientData, targetCount
}

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

        fmt.Fprintf(conn, "UserID: %s\n", clientData[i].UserID)
        for _, rating := range clientData[i].Data {
            fmt.Fprintf(conn, "%s,%s,%.2f\n", rating.UserID, rating.MovieID, rating.Rating)
        }
        fmt.Printf("Datos enviados a %s\n", clienteIP)
    }
}

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

    actualizarTopGlobal(tempResults)
}

func actualizarTopGlobal(tempResults []Recommendation) {
    muTopGlobal.Lock()
    defer muTopGlobal.Unlock()

    ratingMap := make(map[string][]float64)

    for _, rec := range topGlobal {
        ratingMap[rec.MovieID] = append(ratingMap[rec.MovieID], rec.Rating)
    }

    for _, rec := range tempResults {
        ratingMap[rec.MovieID] = append(ratingMap[rec.MovieID], rec.Rating)
    }

    newTopGlobal := []Recommendation{}
    for movieID, ratings := range ratingMap {
        newTopGlobal = append(newTopGlobal, Recommendation{
            MovieID: movieID,
            Rating:  promedio(ratings),
        })
    }

    sort.Slice(newTopGlobal, func(i, j int) bool {
        return newTopGlobal[i].Rating > newTopGlobal[j].Rating
    })

    if len(newTopGlobal) > 15 {
        newTopGlobal = newTopGlobal[:15]
    }

    topGlobal = newTopGlobal
    fmt.Println("Top Global actualizado:", topGlobal)
}

func calcularTop3Final() []Recommendation {
    muTopGlobal.Lock()
    defer muTopGlobal.Unlock()

    if len(topGlobal) > 3 {
        topGlobal = topGlobal[:3]

        seenMovies := make(map[string][]float64)
        for _, rec := range topGlobal {
            seenMovies[rec.MovieID] = append(seenMovies[rec.MovieID], rec.Rating)
        }

        for i, rec := range topGlobal {
            if len(seenMovies[rec.MovieID]) > 1 {
                topGlobal[i].Rating = promedio(seenMovies[rec.MovieID])
            }
        }
    }

    fmt.Println("Top 3 final:")
    for _, rec := range topGlobal {
        fmt.Printf("MovieID: %s, Predicted Rating: %.2f\n", rec.MovieID, rec.Rating)
    }

    return topGlobal
}

func promedio(nums []float64) float64 {
    sum := 0.0
    for _, num := range nums {
        sum += num
    }
    return sum / float64(len(nums))
}
