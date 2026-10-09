// Package product is the one home of Visitron's name and the names of the
// files it keeps, so a rename touches one file.
package product

const (
	// Name is the product's name as every surface shows it.
	Name = "Visitron"
	// Author is credited in About (FR-072).
	Author = "Oliver Ernster"
	// RecordFileName is the SQLite file under the data folder (C-3).
	RecordFileName = "visitron.db"
	// LogFileName is the run log under the data folder (NFR-OBS-001).
	LogFileName = "Log.txt"
	// UniqueID keeps Visitron to one running copy (FR-053).
	UniqueID = "uk.codecrafter.visitron"
	// Owner is the GitHub account Visitron's own releases live under.
	Owner = "oernster"
	// GoatCounterSite is the owner's one GoatCounter account (A-1).
	GoatCounterSite = "https://oernster.goatcounter.com"
	// DonateURL is what the donate button opens (FR-073).
	DonateURL = "https://www.paypal.com/ncp/payment/NRXS4SP24A6C8"
)
