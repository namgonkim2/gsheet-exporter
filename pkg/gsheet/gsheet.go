package gsheet

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/gsheet-exporter/pkg/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Gsheet struct {
	GoogleCredentials string
	SpreadsheetId     string
	ReadRange         string

	Service *sheets.Service
}

func NewGsheet(envs map[string]string) (*Gsheet, error) {
	logger := logger.GetInstance()

	ctx := context.Background()
	b, err := ioutil.ReadFile(envs["GOOGLE_APPLICATION_CREDENTIALS"])
	if err != nil {
		logger.Error.Printf("Unable to read client secret file: %v", err)
		return nil, err
	}
	config, err := google.JWTConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		logger.Error.Printf("Unable to parse client secret file to config: %v", err)
		return nil, err
	}
	client := config.Client(oauth2.NoContext)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		logger.Error.Printf("Unable to retrieve Sheets client: %v", err)
		return nil, err
	}

	gsh := &Gsheet{
		GoogleCredentials: envs["GOOGLE_APPLICATION_CREDENTIALS"],
		SpreadsheetId:     envs["TARGET_SHEETS"],
		ReadRange:         envs["TARGET_SHEETS_RANGE"],

		Service: srv,
	}

	return gsh, nil
}

func openSheet(envs map[string]string) (*sheets.Service, error) {

	return nil, nil
}

func SheetRead(envs map[string]string) ([]string, error) {

	srv, _ := openSheet(envs)

	// Prints the names and majors of students in a sample spreadsheet:
	// https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
	spreadsheetId := envs["TARGET_SHEETS"]
	readRange := strings.Split(envs["TARGET_SHEETS_RANGE"], ",")
	list := []string{}

	for _, readRangeValue := range readRange {
		resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRangeValue).Do()
		if err != nil {
			return []string{}, err
			// log.Fatalf("Unable to retrieve data from sheet: %v", err)
		}

		if len(resp.Values) == 0 {
			fmt.Println("No data found.")
		} else {
			for _, row := range resp.Values {
				var data = ""
				var except = ""
				// Print columns A and E, which correspond to indices row[0] and row[4].
				// C와 D컬럼의 row 데이터 리스트 중 빈칸이 있으면 pass
				if len(row) > 0 {
					// 길이가 2 이상인 경우 -> [이미지명 export값] 보유
					if len(row) > 1 {
						except = fmt.Sprint(row[1])
						// fmt.Println(except)
						if except != "FALSE" { // export가 false가 아닌 이미지 모두 listup
							data = fmt.Sprint(row[0])
							fmt.Println(data)
							list = append(list, data)
						}
					} else { // export가 없으면 true가 default, image listup
						data = fmt.Sprint(row[0])
						list = append(list, data)
					}

				}
			}
		}
	}

	return list, nil
}

func SheetWrite(envs map[string]string) (string, error) {

	srv, err := openSheet(envs)

	// Prints the names and majors of students in a sample spreadsheet:
	// https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
	spreadsheetId := envs["TARGET_SHEETS"]
	writeRange := "unsupported!B8"
	values := []interface{}{"test write"}

	var vr sheets.ValueRange
	vr.Values = append(vr.Values, values)
	resp, err := srv.Spreadsheets.Values.Append(spreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()

	result := "OK"

	if err != nil {
		fmt.Printf("Unable to append data to sheet: %v", err)
	}
	fmt.Println("spreadsheet push ", resp.Header)

	return result, err
}
