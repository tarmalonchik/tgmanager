//go:generate go-enum -f=$GOFILE --nocase
package tgmanager

// CallBackAppearType ENUM(update,resend,resend_delete_old)
type CallBackAppearType string
