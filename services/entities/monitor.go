package entities

import (
	"log"
	"time"
)

// MonitorHighValueAddresses theo dõi tất cả các địa chỉ có giá trị cao
func MonitorHighValueAddresses() {
	wg.Add(1)
	defer wg.Done()

	log.Println("[BẮT ĐẦU] Theo dõi các địa chỉ có giá trị cao...")

	// Tạo ticker để kiểm tra định kỳ các địa chỉ mới
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Hàm để thêm các địa chỉ vào danh sách theo dõi
	addAddressesToMonitoring := func() {
		// Theo dõi các địa chỉ sàn giao dịch
		for addr, info := range exchangeAddresses {
			if !IsAddressMonitored(addr) {
				log.Printf("[THEO DÕI] Thêm địa chỉ sàn giao dịch: %s (%.8f BTCB, %.2f%%)", 
					addr, info.Balance, info.Percentage)
				AddAddressToMonitoring(addr)
			}
		}

		// Theo dõi các địa chỉ tổ chức
		for addr, info := range organizationAddresses {
			if !IsAddressMonitored(addr) {
				log.Printf("[THEO DÕI] Thêm địa chỉ tổ chức: %s (%.8f BTCB, %.2f%%)", 
					addr, info.Balance, info.Percentage)
				AddAddressToMonitoring(addr)
			}
		}

		// Theo dõi các địa chỉ có số dư lớn khác
		for addr, info := range highBalanceAddresses {
			if !IsAddressMonitored(addr) && 
				exchangeAddresses[addr].Balance == 0 && 
				organizationAddresses[addr].Balance == 0 {
				log.Printf("[THEO DÕI] Thêm địa chỉ có số dư lớn: %s (%.8f BTCB, %.2f%%)", 
					addr, info.Balance, info.Percentage)
				AddAddressToMonitoring(addr)
			}
		}
	}
	
	// Thêm các địa chỉ hiện có vào danh sách theo dõi
	addAddressesToMonitoring()

	// Vòng lặp chính để theo dõi
	for {
		select {
		case <-stopChan:
			log.Println("[DỪNG] Dừng theo dõi các địa chỉ có giá trị cao")
			return
		case <-ticker.C:
			// Kiểm tra và thêm các địa chỉ mới vào danh sách theo dõi
			addAddressesToMonitoring()
		}
	}
}
