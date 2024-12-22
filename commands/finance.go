// Package commands berisi implementasi command-command bot Discord
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	// Konstanta untuk warna embed
	colorSuccess = 0x00FF00 // Hijau
	colorError   = 0xFF0000 // Merah
	colorWarning = 0xFF9900 // Orange
	colorInfo    = 0x3498DB // Biru

	// Format waktu default
	timeFormat = "02 Jan 2006 15:04"

	// Path file data
	dataFile = "data.json"

	// Jumlah transaksi per halaman
	transactionsPerPage = 5
)

// Daftar role yang diizinkan menggunakan bot
var AllowedRoles = []string{
	"1041348750540550215",
	"1041350379318820975",
	"1041350844509077524",
	"1041351508417056900",
}

// Transaction merepresentasikan satu transaksi keuangan
type Transaction struct {
	Type      string    `json:"type"`      // "masuk" atau "keluar"
	Amount    float64   `json:"amount"`    // Jumlah uang
	Timestamp time.Time `json:"timestamp"` // Waktu transaksi
}

// FinanceData adalah struktur utama untuk menyimpan data keuangan
type FinanceData struct {
	Transactions []Transaction `json:"transactions"`
}

// hasAllowedRole mengecek apakah user memiliki role yang diizinkan
func hasAllowedRole(member *discordgo.Member) bool {
	if member == nil {
		return false
	}

	for _, role := range member.Roles {
		for _, allowedRole := range AllowedRoles {
			if role == allowedRole {
				return true
			}
		}
	}
	return false
}

// formatRupiah memformat angka ke format mata uang Rupiah
// Contoh: 10000 -> Rp10.000,00
func formatRupiah(amount float64) string {
	str := fmt.Sprintf("%.2f", amount)
	parts := strings.Split(str, ".")

	// Format bagian integer dengan pemisah ribuan
	integer := parts[0]
	length := len(integer)
	var result []rune

	for i, r := range integer {
		if i != 0 && (length-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, r)
	}

	return fmt.Sprintf("Rp%s,%s", string(result), parts[1])
}

// createBaseEmbed membuat template dasar untuk embed message
func createBaseEmbed(title string, color int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:     title,
		Color:     color,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "DevTest Finance Bot",
		},
	}
}

// createTransactionEmbed membuat embed untuk transaksi
func createTransactionEmbed(transType string, amount float64) *discordgo.MessageEmbed {
	var (
		title     string
		color     int
		thumbnail string
	)

	switch transType {
	case "masuk":
		title = "💰 Uang Masuk"
		color = colorSuccess
		thumbnail = "https://cdn-icons-png.flaticon.com/512/2489/2489756.png"
	case "keluar":
		title = "💸 Uang Keluar"
		color = 0xFF6B6B
		thumbnail = "https://cdn-icons-png.flaticon.com/512/2489/2489757.png"
	}

	embed := createBaseEmbed(title, color)
	embed.Description = fmt.Sprintf("**Jumlah:** %s", formatRupiah(amount))
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: thumbnail}

	return embed
}

// createErrorEmbed membuat embed untuk pesan error
func createErrorEmbed(title, description string) *discordgo.MessageEmbed {
	embed := createBaseEmbed(title, colorError)
	embed.Description = description
	return embed
}

// createSuccessEmbed membuat embed untuk pesan sukses
func createSuccessEmbed(title, description string) *discordgo.MessageEmbed {
	embed := createBaseEmbed(title, colorSuccess)
	embed.Description = description
	return embed
}

// createSummaryEmbed membuat embed untuk ringkasan keuangan
func createSummaryEmbed(total float64, transactions []Transaction, page int) *discordgo.MessageEmbed {
	embed := createBaseEmbed("📊 Ringkasan Keuangan", colorInfo)
	
	// Hitung total transaksi masuk dan keluar
	var totalMasuk, totalKeluar float64
	for _, t := range transactions {
		if t.Type == "masuk" {
			totalMasuk += t.Amount
		} else {
			totalKeluar += t.Amount
		}
	}

	// Hitung total halaman
	totalPages := (len(transactions) + transactionsPerPage - 1) / transactionsPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	// Validasi halaman
	if page < 1 {
		page = 1
	} else if page > totalPages {
		page = totalPages
	}

	// Format deskripsi dengan informasi lengkap
	description := fmt.Sprintf("**💰 Total Saldo:** %s\n", formatRupiah(total))
	description += fmt.Sprintf("**📥 Total Masuk:** %s\n", formatRupiah(totalMasuk))
	description += fmt.Sprintf("**📤 Total Keluar:** %s\n", formatRupiah(totalKeluar))
	description += fmt.Sprintf("**📝 Jumlah Transaksi:** %d\n", len(transactions))
	description += fmt.Sprintf("**📄 Halaman:** %d dari %d", page, totalPages)
	
	embed.Description = description

	if len(transactions) > 0 {
		// Hitung indeks awal dan akhir untuk halaman yang diminta
		start := len(transactions) - (page * transactionsPerPage)
		end := start + transactionsPerPage
		
		if start < 0 {
			start = 0
		}
		if end > len(transactions) {
			end = len(transactions)
		}

		// Format daftar transaksi
		var transactionList strings.Builder
		transactionList.WriteString("```\n")

		for i := start; i < end; i++ {
			t := transactions[i]
			symbol := "➕"
			if t.Type == "keluar" {
				symbol = "➖"
			}
			fmt.Fprintf(&transactionList, "%s %s (%s)\n",
				symbol,
				formatRupiah(t.Amount),
				t.Timestamp.Format(timeFormat))
		}
		
		if totalPages > 1 {
			transactionList.WriteString(fmt.Sprintf("\n💡 Gunakan ?totaluang <1-%d> untuk melihat halaman lain", totalPages))
		}
		transactionList.WriteString("```")

		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   fmt.Sprintf("📝 Riwayat Transaksi (Halaman %d/%d)", page, totalPages),
				Value:  transactionList.String(),
				Inline: false,
			},
		}
	}

	return embed
}

// createConfirmationEmbed membuat embed untuk konfirmasi penghapusan
func createConfirmationEmbed() *discordgo.MessageEmbed {
	embed := createBaseEmbed("⚠️ Konfirmasi Penghapusan", colorWarning)
	embed.Description = "Apakah Anda yakin ingin menghapus **SEMUA** transaksi?\n" +
		"Data yang sudah dihapus **TIDAK DAPAT** dikembalikan!\n\n" +
		"Ketik `?confirmclear` untuk mengkonfirmasi penghapusan."
	embed.Footer.Text = "Peringatan: Aksi ini tidak dapat dibatalkan!"
	return embed
}

// loadFinanceData membaca dan parse data keuangan dari file JSON
func loadFinanceData() (*FinanceData, error) {
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return &FinanceData{}, err
	}

	var financeData FinanceData
	if err := json.Unmarshal(data, &financeData); err != nil {
		return &FinanceData{}, err
	}

	return &financeData, nil
}

// saveFinanceData menyimpan data keuangan ke file JSON
func saveFinanceData(data *FinanceData) error {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(dataFile, jsonData, 0644)
}

// handleTransaction menangani proses transaksi (masuk/keluar)
func handleTransaction(s *discordgo.Session, m *discordgo.MessageCreate, transType string, amount float64) {
	financeData, err := loadFinanceData()
	if err != nil {
		s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed(
			"Error",
			"Gagal memuat data: "+err.Error(),
		), m.Reference())
		return
	}

	transaction := Transaction{
		Type:      transType,
		Amount:    amount,
		Timestamp: time.Now(),
	}

	financeData.Transactions = append(financeData.Transactions, transaction)
	if err := saveFinanceData(financeData); err != nil {
		s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed(
			"Error",
			"Gagal menyimpan data: "+err.Error(),
		), m.Reference())
		return
	}

	s.ChannelMessageSendEmbedReply(m.ChannelID, createTransactionEmbed(transType, amount), m.Reference())
}

// calculateTotal menghitung total saldo dari semua transaksi
func calculateTotal(transactions []Transaction) float64 {
	var total float64
	for _, t := range transactions {
		if t.Type == "masuk" {
			total += t.Amount
		} else {
			total -= t.Amount
		}
	}
	return total
}

// HandleFinanceCommand menangani semua command keuangan
func HandleFinanceCommand(s *discordgo.Session, m *discordgo.MessageCreate, command string, args []string) {
	// Verifikasi role user
	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed(
			"Error",
			"Gagal memverifikasi role: "+err.Error(),
		), m.Reference())
		return
	}

	if !hasAllowedRole(member) {
		s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed(
			"⛔ Akses Ditolak",
			"Anda tidak memiliki izin untuk menggunakan command ini.",
		), m.Reference())
		return
	}

	switch command {
	case "uangmasuk", "uangkeluar":
		if len(args) != 2 {
			s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed(
				"Format Salah",
				fmt.Sprintf("Gunakan: ?%s <jumlah>", command),
			), m.Reference())
			return
		}

		amount, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed(
				"Input Tidak Valid",
				"Jumlah harus berupa angka",
			), m.Reference())
			return
		}

		handleTransaction(s, m, strings.TrimPrefix(command, "uang"), amount)

	case "totaluang":
		financeData, err := loadFinanceData()
		if err != nil {
			s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed(
				"Error",
				"Gagal memuat data: "+err.Error(),
			), m.Reference())
			return
		}

		// Parse halaman dari argumen
		page := 1
		if len(args) > 1 {
			if p, err := strconv.Atoi(args[1]); err == nil {
				page = p
			}
		}

		total := calculateTotal(financeData.Transactions)
		s.ChannelMessageSendEmbedReply(m.ChannelID,
			createSummaryEmbed(total, financeData.Transactions, page),
			m.Reference())

	case "cleartransaksi":
		s.ChannelMessageSendEmbedReply(m.ChannelID, createConfirmationEmbed(), m.Reference())

	case "confirmclear":
		if err := saveFinanceData(&FinanceData{}); err != nil {
			s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed(
				"Error",
				"Gagal menghapus data: "+err.Error(),
			), m.Reference())
			return
		}

		s.ChannelMessageSendEmbedReply(m.ChannelID, createSuccessEmbed(
			"🗑️ Data Berhasil Dihapus",
			"Semua transaksi telah dihapus dari sistem.",
		), m.Reference())
	}
}
