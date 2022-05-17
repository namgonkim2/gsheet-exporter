package gsheet

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
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

var (
	imageList       []string // all image list
	exceptImageList []string // export false image list
)

func NewGsheet() (*Gsheet, error) {

	logger := logger.GetInstance()

	b, err := ioutil.ReadFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
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

	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		logger.Error.Printf("Unable to retrieve Sheets client: %v", err)
		return nil, err
	}

	return &Gsheet{
		GoogleCredentials: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		SpreadsheetId:     os.Getenv("TARGET_SHEETS"),
		ReadRange:         os.Getenv("TARGET_SHEETS_RANGE"),

		Service: srv,
	}, nil
}

func (gsheet *Gsheet) GetGsheet() ([]string, []string, error) {

	logger := logger.GetInstance()

	// Instance information set
	spreadsheetId := gsheet.SpreadsheetId // envs["TARGET_SHEETS"]
	readRange := strings.Split(gsheet.ReadRange, ",")
	srv := gsheet.Service

	// clear
	imageList = []string{}
	exceptImageList = []string{}

	for _, readRangeValue := range readRange {
		resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRangeValue).Do()
		if err != nil {
			logger.Error.Printf("Unable to retrieve data from sheet: %v", err)
			return nil, nil, err
		}

		// google sheet read, then build image list func
		parseImageList(resp)
	}
	return imageList, exceptImageList, nil
}

func (gsheet *Gsheet) SetGsheet() (string, error) {

	logger := logger.GetInstance()

	// Instance information set
	spreadsheetId := gsheet.SpreadsheetId // envs["TARGET_SHEETS"]
	writeRange := "unsupported!B8"
	srv := gsheet.Service

	values := []interface{}{"test write"}

	var vr sheets.ValueRange
	vr.Values = append(vr.Values, values)
	resp, err := srv.Spreadsheets.Values.Append(spreadsheetId, writeRange, &vr).ValueInputOption("RAW").Do()

	result := "OK"

	if err != nil {
		logger.Error.Printf("Unable to append data to sheet: %v", err)
	}
	logger.Info.Println("spreadsheet push ", resp.Header)

	return result, err
}

func parseImageList(resp *sheets.ValueRange) {

	logger := logger.GetInstance()

	if len(resp.Values) == 0 {
		logger.Error.Println("No data found.")
	} else {
		for _, row := range resp.Values {
			data := ""
			except := ""
			// Print columns A and E, which correspond to indices row[0] and row[4].
			// C와 D컬럼의 row 데이터 리스트 중 빈칸이 있으면 pass
			if len(row) > 0 {
				data = fmt.Sprint(row[0])
				// 길이가 2 이상인 경우 -> [이미지명 export값] 보유
				if len(row) > 1 {
					except = fmt.Sprint(row[1])
					if except == "FALSE" { // export가 false가 아닌 이미지 모두 listup
						exceptImageList = append(exceptImageList, data)
					} else {
						imageList = append(imageList, data)
					}
				} else { // export가 없으면 true가 default, image listup
					imageList = append(imageList, data)
				}

			}
		}
	}
}
