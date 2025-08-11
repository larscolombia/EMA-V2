package countries

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// Country is the response model matching CountryModel in Flutter
// { id, name, short_code, phone_code }

type Country struct {
    ID        int    `json:"id"`
    Name      string `json:"name"`
    ShortCode string `json:"short_code"`
    PhoneCode int    `json:"phone_code"`
}

// RegisterRoutes registers GET /countries with a static minimal list for now.
func RegisterRoutes(r *gin.Engine) {
    r.GET("/countries", func(c *gin.Context) {
        // Minimal seed list; extend as needed or read from DB.
        data := []Country{
            {ID: 1, Name: "Colombia", ShortCode: "CO", PhoneCode: 57},
            {ID: 2, Name: "México", ShortCode: "MX", PhoneCode: 52},
            {ID: 3, Name: "Perú", ShortCode: "PE", PhoneCode: 51},
            {ID: 4, Name: "Argentina", ShortCode: "AR", PhoneCode: 54},
            {ID: 5, Name: "Chile", ShortCode: "CL", PhoneCode: 56},
            {ID: 6, Name: "España", ShortCode: "ES", PhoneCode: 34},
            {ID: 7, Name: "Estados Unidos", ShortCode: "US", PhoneCode: 1},
        }
        c.JSON(http.StatusOK, data)
    })
}
