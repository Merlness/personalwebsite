package config

import "os"

func ResolvePortfolioRoot() string {
	optimized := "content/portfolio_optimized"
	if _, err := os.Stat(optimized); err == nil {
		return optimized
	}
	return "content/portfolio"
}

func ResolveAboutmeRoot() string {
	optimized := "content/aboutme_optimized"
	if _, err := os.Stat(optimized); err == nil {
		return optimized
	}
	return "content/aboutme"
}
