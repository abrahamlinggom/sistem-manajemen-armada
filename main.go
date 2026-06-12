package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Struktur database
type VehicleLocation struct {
	ID        uint    `gorm:"primaryKey" json:"-"`
	VehicleID string  `json:"vehicle_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

// table vehicle_location
func (VehicleLocation) TableName() string {
	return "vehicle_locations"
}

var db *gorm.DB
var rabbitConn *amqp.Connection

func main() {
	// koneksi ke postgresql
	dsn := getEnv("DB_DSN", "host=localhost user=postgres password=Ghizooo3 dbname=armada port=5432 sslmode=disable")
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal koneksi database:", err)
	}
	// melakukan migration table
	db.AutoMigrate(&VehicleLocation{})

	// koneksi ke rabbitmq
	rabbitConn, err = amqp.Dial(getEnv("RABBIT_URL", "amqp://guest:guest@localhost:5672/"))
	if err != nil {
		log.Println("RabbitMQ belum merespons")
	}

	// menjalankan rabbitmq di latar belakang
	go workerRabbitMQ()

	// koneksi ke mqtt
	opts := mqtt.NewClientOptions().AddBroker(getEnv("MQTT_BROKER", "tcp://localhost:1883")).SetClientID("backend_utama")
	// kalau ada data masuk
	opts.SetDefaultPublishHandler(terimaDataDariArmada)

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("Gagal koneksi MQTT:", token.Error())
	}
	mqttClient.Subscribe("/fleet/vehicle/+/location", 1, nil) // subscribe ke topik MQTT untuk menerima data lokasi dari armada

	// Membuat rest API dengan gin
	r := gin.Default()
	r.GET("/vehicles/:vehicle_id/location", apiLokasiTerakhir)
	r.GET("/vehicles/:vehicle_id/history", apiRiwayatLokasi)

	fmt.Println("Backend berjalan di port 8080")
	r.Run(":8080")
}

// fungsi untuk menerima data dari armada melalui mqtt, menyimpan ke database, dan memeriksa geofence untuk mengirim peringatan jika diperlukan
var terimaDataDariArmada mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var loc VehicleLocation
	json.Unmarshal(msg.Payload(), &loc)

	if loc.VehicleID == "" {
		return
	}

	// Simpan ke PostgreSQL
	db.Create(&loc)
	fmt.Printf("Tersimpan ke DB: Armada %s di Kordinat %f, %f\n", loc.VehicleID, loc.Latitude, loc.Longitude)

	// Hitung jarak
	jarak := hitungJarak(loc.Latitude, loc.Longitude, -6.1754, 106.8272)
	if jarak <= 50 {
		kirimPeringatanGeofence(loc)
	}
}

// Menerbitkan pengumuman ke RabbitMQ
func kirimPeringatanGeofence(loc VehicleLocation) {
	if rabbitConn == nil {
		return
	}
	ch, _ := rabbitConn.Channel()
	defer ch.Close()

	ch.ExchangeDeclare("fleet.events", "direct", true, false, false, false, nil)

	pesanGeofence := map[string]interface{}{
		"vehicle_id": loc.VehicleID,
		"event":      "geofence_entry",
		"location":   map[string]float64{"latitude": loc.Latitude, "longitude": loc.Longitude},
		"timestamp":  loc.Timestamp,
	}
	body, _ := json.Marshal(pesanGeofence)
	ch.Publish("fleet.events", "geofence_alerts", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// Membaca RabbitMQ
func workerRabbitMQ() {
	if rabbitConn == nil {
		return
	}
	ch, _ := rabbitConn.Channel()
	defer ch.Close()

	q, _ := ch.QueueDeclare("geofence_alerts", true, false, false, false, nil)
	msgs, _ := ch.Consume(q.Name, "", true, false, false, false, nil)

	for d := range msgs {
		fmt.Printf("[GEOFENCE ALERT] %s\n", d.Body)
	}
}

// Controller API
func apiLokasiTerakhir(c *gin.Context) {
	vehicleID := c.Param("vehicle_id")
	var loc VehicleLocation
	db.Where("vehicle_id = ?", vehicleID).Order("timestamp desc").First(&loc)
	c.JSON(http.StatusOK, loc)
}

func apiRiwayatLokasi(c *gin.Context) {
	vehicleID := c.Param("vehicle_id")
	start := c.Query("start")
	end := c.Query("end")
	var history []VehicleLocation
	db.Where("vehicle_id = ? AND timestamp >= ? AND timestamp <= ?", vehicleID, start, end).Find(&history)
	c.JSON(http.StatusOK, history)
}

// Rumus mencari jarak
func hitungJarak(lat1, lon1, lat2, lon2 float64) float64 {
	p := 0.017453292519943295 // Math.PI / 180
	a := 0.5 - math.Cos((lat2-lat1)*p)/2 + math.Cos(lat1*p)*math.Cos(lat2*p)*(1-math.Cos((lon2-lon1)*p))/2
	return 12742000 * math.Asin(math.Sqrt(a)) // 2 * R (Bumi) * asin...
}

func getEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}