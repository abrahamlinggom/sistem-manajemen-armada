package main

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("mobil_b1234xyz")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// titik awal
	lat := -6.1758
	lon := 106.8272

	fmt.Println("Armada berjalan")

	for {
		lat += 0.0001 // Bergerak mendekat titik -6.1754

		data := map[string]interface{}{
			"vehicle_id": "B1234XYZ",
			"latitude":   lat,
			"longitude":  lon,
			"timestamp":  time.Now().Unix(),
		}

		payload, _ := json.Marshal(data)
		client.Publish("/fleet/vehicle/B1234XYZ/location", 0, false, payload)

		fmt.Printf("Koordinat dikirim via MQTT\n")
		time.Sleep(2 * time.Second) // setiap 2 detik kirim data lokasi baru
	}
}
