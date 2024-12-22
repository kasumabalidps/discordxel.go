package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type Transaction struct {
	Type      string    `json:"type"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

type FinanceData struct {
	Transactions []Transaction `json:"transactions"`
}

// Helper functions untuk membuat embed messages
func createErrorEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xFF0000, // Merah untuk error
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func createSuccessEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00FF00, // Hijau untuk sukses
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Helper function untuk format uang Indonesia
func formatRupiah(amount float64) string {
	// Konversi ke string dengan 2 desimal
	str := fmt.Sprintf("%.2f", amount)

	// Pisahkan bagian desimal
	parts := strings.Split(str, ".")

	// Format ribuan untuk bagian integer
	integer := parts[0]
	length := len(integer)
	var result []rune

	for i, r := range integer {
		if i != 0 && (length-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, r)
	}

	// Gabungkan dengan bagian desimal, ganti titik dengan koma
	return fmt.Sprintf("Rp%s,%s", string(result), parts[1])
}

func createTransactionEmbed(transType string, amount float64) *discordgo.MessageEmbed {
	var (
		title     string
		color     int
		thumbnail string
	)

	if transType == "masuk" {
		title = "💰 Uang Masuk"
		color = 0x00FF00
		thumbnail = "https://cdn-icons-png.flaticon.com/512/2489/2489756.png"
	} else {
		title = "💸 Uang Keluar"
		color = 0xFF6B6B
		thumbnail = "https://cdn-icons-png.flaticon.com/512/2489/2489757.png"
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("**Jumlah:** %s", formatRupiah(amount)),
		Color:       color,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: thumbnail,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "DevTest Finance Bot",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func createSummaryEmbed(total float64, transactions []Transaction) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "📊 Ringkasan Keuangan",
		Description: fmt.Sprintf("**Total Saldo:** %s", formatRupiah(total)),
		Color:       0x3498DB,
		Fields:      []*discordgo.MessageEmbedField{},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "DevTest Finance Bot",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Tambahkan field untuk 5 transaksi terakhir
	if len(transactions) > 0 {
		transactionList := "```\n"
		start := len(transactions) - 5
		if start < 0 {
			start = 0
		}

		for i := start; i < len(transactions); i++ {
			t := transactions[i]
			symbol := "➕"
			if t.Type == "keluar" {
				symbol = "➖"
			}
			transactionList += fmt.Sprintf("%s %s (%s)\n",
				symbol,
				formatRupiah(t.Amount),
				t.Timestamp.Format("02 Jan 2006 15:04"))
		}
		transactionList += "```"

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📝 Riwayat Transaksi Terakhir",
			Value:  transactionList,
			Inline: false,
		})
	}

	return embed
}

func createConfirmationEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⚠️ Konfirmasi Penghapusan",
		Description: "Apakah Anda yakin ingin menghapus **SEMUA** transaksi?\nData yang sudah dihapus **TIDAK DAPAT** dikembalikan!\n\nKetik `?confirmclear` untuk mengkonfirmasi penghapusan.",
		Color:       0xFF9900, // Orange untuk warning
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Peringatan: Aksi ini tidak dapat dibatalkan!",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func loadFinanceData() (*FinanceData, error) {
	data, err := os.ReadFile("data.json")
	if err != nil {
		return &FinanceData{}, err
	}

	var financeData FinanceData
	err = json.Unmarshal(data, &financeData)
	if err != nil {
		return &FinanceData{}, err
	}

	return &financeData, nil
}

func saveFinanceData(data *FinanceData) error {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile("data.json", jsonData, 0644)
}

func handleTransaction(s *discordgo.Session, m *discordgo.MessageCreate, transType string, amount float64) {
	financeData, err := loadFinanceData()
	if err != nil {
		s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed("Error", "Gagal memuat data: "+err.Error()), m.Reference())
		return
	}

	transaction := Transaction{
		Type:      transType,
		Amount:    amount,
		Timestamp: time.Now(),
	}

	financeData.Transactions = append(financeData.Transactions, transaction)
	if err := saveFinanceData(financeData); err != nil {
		s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed("Error", "Gagal menyimpan data: "+err.Error()), m.Reference())
		return
	}

	s.ChannelMessageSendEmbedReply(m.ChannelID, createTransactionEmbed(transType, amount), m.Reference())
}

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}
}

func main() {
	// Mengambil token dari .env
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN tidak ditemukan di .env")
	}

	log.Println("Starting bot...")

	discord, err := discordgo.New("Bot " + token)
	discord.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	if err != nil {
		log.Fatal(err)
	}

	discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot || !strings.HasPrefix(m.Content, "?") {
			return
		}

		args := strings.Fields(m.Content[1:])
		if len(args) < 1 {
			return
		}

		command := strings.ToLower(args[0])

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

			transType := "masuk"
			if command == "uangkeluar" {
				transType = "keluar"
			}

			handleTransaction(s, m, transType, amount)

		case "totaluang":
			financeData, err := loadFinanceData()
			if err != nil {
				s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed("Error", "Gagal memuat data: "+err.Error()), m.Reference())
				return
			}

			var total float64
			for _, t := range financeData.Transactions {
				if t.Type == "masuk" {
					total += t.Amount
				} else {
					total -= t.Amount
				}
			}

			s.ChannelMessageSendEmbedReply(m.ChannelID, createSummaryEmbed(total, financeData.Transactions), m.Reference())

		case "cleartransaksi":
			s.ChannelMessageSendEmbedReply(m.ChannelID, createConfirmationEmbed(), m.Reference())

		case "confirmclear":
			// Reset data.json ke array kosong
			emptyData := &FinanceData{
				Transactions: []Transaction{},
			}

			if err := saveFinanceData(emptyData); err != nil {
				s.ChannelMessageSendEmbedReply(m.ChannelID, createErrorEmbed(
					"Error",
					"Gagal menghapus data: "+err.Error(),
				), m.Reference())
				return
			}

			successEmbed := createSuccessEmbed(
				"🗑️ Data Berhasil Dihapus",
				"Semua transaksi telah dihapus dari sistem.",
			)
			s.ChannelMessageSendEmbedReply(m.ChannelID, successEmbed, m.Reference())
		}
	})

	err = discord.Open()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Bot is now running perfectly boss.")
	select {}
}
