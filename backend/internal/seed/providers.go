package seed

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/dimitris-taskou/cost-calculator/internal/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var providers = []struct {
	id   string
	data models.Provider
}{
	// Energy
	{"dei", models.Provider{Name: "ΔΕΗ", Category: "energy", PaymentURL: "https://mydei.dei.gr/el/login/", Color: "#0066CC", Description: "Δημόσια Επιχείρηση Ηλεκτρισμού"}},
	{"heron", models.Provider{Name: "Ήρων", Category: "energy", PaymentURL: "https://mypayment.heron.gr/", Color: "#E8531B", Description: "Ήρων Θερμοηλεκτρική"}},
	{"zenith", models.Provider{Name: "Ζενίθ", Category: "energy", PaymentURL: "https://zenith.gr/el/pliromi-logariasmou/", Color: "#6B21A8", Description: "ZeniΘ Energy"}},
	{"protergia", models.Provider{Name: "Protergia", Category: "energy", PaymentURL: "https://www.protergia.gr/", Color: "#F59E0B", Description: "Protergia"}},
	// Water
	{"eydap", models.Provider{Name: "ΕΥΔΑΠ", Category: "water", PaymentURL: "https://www.eydap.gr/en/MyAccount/EservicesInfo/", Color: "#0EA5E9", Description: "Εταιρεία Υδρεύσεως Αποχετεύσεως Πρωτευούσης"}},
	{"eyath", models.Provider{Name: "ΕΥΑΘ", Category: "water", PaymentURL: "https://www.eyath.gr/", Color: "#06B6D4", Description: "Εταιρεία Ύδρευσης Αποχέτευσης Θεσσαλονίκης"}},
	// Telecom
	{"vodafone", models.Provider{Name: "Vodafone", Category: "telecom", PaymentURL: "https://www.vodafone.gr/myvodafone/", Color: "#E60000", Description: "Vodafone Greece"}},
	{"cosmote", models.Provider{Name: "Cosmote", Category: "telecom", PaymentURL: "https://my.cosmote.gr/selfcare/jsp/pay_ebill.jsp", Color: "#00A650", Description: "Cosmote Mobile Telecommunications"}},
	{"nova", models.Provider{Name: "Nova", Category: "telecom", PaymentURL: "https://billpay.nova.gr/", Color: "#FF6B00", Description: "Nova (Wind Hellas)"}},
	{"wind", models.Provider{Name: "Wind", Category: "telecom", PaymentURL: "https://www.wind.gr/", Color: "#0047AB", Description: "Wind Hellas"}},
	// Streaming
	{"netflix", models.Provider{Name: "Netflix", Category: "streaming", PaymentURL: "https://www.netflix.com/gr/", Color: "#E50914", Description: "Netflix"}},
	{"spotify", models.Provider{Name: "Spotify", Category: "streaming", PaymentURL: "https://www.spotify.com/gr/", Color: "#1DB954", Description: "Spotify"}},
	{"youtube-premium", models.Provider{Name: "YouTube Premium", Category: "streaming", PaymentURL: "https://www.youtube.com/premium", Color: "#FF0000", Description: "YouTube Premium"}},
	{"disney-plus", models.Provider{Name: "Disney+", Category: "streaming", PaymentURL: "https://www.disneyplus.com/", Color: "#113CCF", Description: "Disney+"}},
	{"apple-tv", models.Provider{Name: "Apple TV+", Category: "streaming", PaymentURL: "https://tv.apple.com/", Color: "#555555", Description: "Apple TV+"}},
	{"amazon-prime", models.Provider{Name: "Amazon Prime", Category: "streaming", PaymentURL: "https://www.amazon.com/prime", Color: "#00A8E0", Description: "Amazon Prime Video"}},
	{"max", models.Provider{Name: "Max", Category: "streaming", PaymentURL: "https://www.max.com/", Color: "#002BE7", Description: "Max (HBO)"}},
	// Subscriptions
	{"microsoft-365", models.Provider{Name: "Microsoft 365", Category: "subscription", PaymentURL: "https://account.microsoft.com/", Color: "#D83B01", Description: "Microsoft 365"}},
	{"google-one", models.Provider{Name: "Google One", Category: "subscription", PaymentURL: "https://one.google.com/", Color: "#4285F4", Description: "Google One"}},
	{"apple-icloud", models.Provider{Name: "iCloud+", Category: "subscription", PaymentURL: "https://www.icloud.com/", Color: "#3B82F6", Description: "Apple iCloud+"}},
	{"adobe", models.Provider{Name: "Adobe CC", Category: "subscription", PaymentURL: "https://account.adobe.com/", Color: "#FF0000", Description: "Adobe Creative Cloud"}},
	{"chatgpt", models.Provider{Name: "ChatGPT Plus", Category: "subscription", PaymentURL: "https://chat.openai.com/", Color: "#10A37F", Description: "OpenAI ChatGPT Plus"}},
	// Insurance / Finance
	{"insurance", models.Provider{Name: "Ασφάλεια", Category: "finance", PaymentURL: "", Color: "#0369A1", Description: "Ασφαλιστήριο"}},
	{"enfia", models.Provider{Name: "ΕΝΦΙΑ", Category: "finance", PaymentURL: "https://www.aade.gr/", Color: "#1D4ED8", Description: "Ενιαίος Φόρος Ιδιοκτησίας Ακινήτων"}},
	// Housing
	{"koinoxrista", models.Provider{Name: "Κοινόχρηστα", Category: "housing", PaymentURL: "", Color: "#059669", Description: "Κοινόχρηστα πολυκατοικίας"}},
	{"enoikio", models.Provider{Name: "Ενοίκιο", Category: "housing", PaymentURL: "", Color: "#7C3AED", Description: "Ενοίκιο κατοικίας"}},
	{"danio", models.Provider{Name: "Δάνειο", Category: "housing", PaymentURL: "", Color: "#DC2626", Description: "Δόση δανείου"}},
	// Car
	{"kteo", models.Provider{Name: "ΚΤΕΟ", Category: "car", PaymentURL: "", Color: "#EA580C", Description: "Τεχνικός Έλεγχος Οχημάτων"}},
	{"car-insurance", models.Provider{Name: "Ασφάλεια Αυτοκινήτου", Category: "car", PaymentURL: "", Color: "#B45309", Description: "Ασφαλιστήριο αυτοκινήτου"}},
	{"car-taxes", models.Provider{Name: "Τέλη Κυκλοφορίας", Category: "car", PaymentURL: "https://www.aade.gr/", Color: "#0369A1", Description: "Ετήσια τέλη κυκλοφορίας"}},
	{"car-service", models.Provider{Name: "Service Αυτοκινήτου", Category: "car", PaymentURL: "", Color: "#374151", Description: "Συντήρηση / επισκευή αυτοκινήτου"}},
}

func Providers(ctx context.Context, fs *firestore.Client) error {
	col := fs.Collection("providers")
	for _, p := range providers {
		ref := col.Doc(p.id)
		_, err := ref.Get(ctx)
		if status.Code(err) == codes.NotFound {
			p.data.ID = p.id
			if _, err := ref.Set(ctx, p.data); err != nil {
				return err
			}
		}
	}
	return nil
}
