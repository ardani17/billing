package usecase

import "golang.org/x/crypto/bcrypt"

// HashPassword menghash password menggunakan bcrypt dengan cost factor yang dikonfigurasi.
// Cost factor minimal bcrypt.MinCost; jika cost yang diberikan lebih kecil dari MinCost,
// maka akan digunakan bcrypt.DefaultCost (10) sesuai standar keamanan.
func HashPassword(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword memverifikasi password terhadap hash bcrypt yang tersimpan.
// Mengembalikan nil jika password cocok, error jika tidak.
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
