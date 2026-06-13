# Sistem Manajemen Armada

## Spesifikasi
1. **Golang**: Layanan backend & API (Gin).
2. **PostgreSQL**: Penyimpanan koordinat armada.
3. **MQTT**: Broker penerima aliran data lokasi kendaraan dengan latensi rendah.
4. **RabbitMQ**: Antrean pesan untuk deteksi Geofence.
5. **Docker**: Kontainer untuk ekosistem.

## Cara Menjalankan
1. Pastikan Docker Desktop sudah berjalan.
2. Clone repositori ini, lalu jalankan:
   ```bash
   docker-compose up -d --build