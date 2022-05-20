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
	WriteRange        string

	Service *sheets.Service
	Ctx     context.Context
}

var (
	log = logger.GetInstance()
)

func NewGsheet(googleCredentials, spreadsheetId, readRange, writeRange string) (*Gsheet, error) {
	b, err := ioutil.ReadFile(googleCredentials)
	if err != nil {
		log.Error.Printf("Unable to read client secret file: %v", err)
		return nil, err
	}
	config, err := google.JWTConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Error.Printf("Unable to parse client secret file to config: %v", err)
		return nil, err
	}
	client := config.Client(oauth2.NoContext)

	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Error.Printf("Unable to retrieve Sheets client: %v", err)
		return nil, err
	}

	return &Gsheet{
		GoogleCredentials: googleCredentials,
		SpreadsheetId:     spreadsheetId,
		ReadRange:         readRange,
		WriteRange:        writeRange,

		Service: srv,
		Ctx:     ctx,
	}, nil
}

// Read image list in google sheet
func (gsheet *Gsheet) GetGsheet() ([]string, []string, error) {

	// Instance information set
	spreadsheetId := gsheet.SpreadsheetId
	readRange := strings.Split(gsheet.ReadRange, ",")
	srv := gsheet.Service

	imageList := []string{}
	exceptImageList := []string{}

	for _, readRangeValue := range readRange {
		resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRangeValue).Do()
		if err != nil {
			log.Error.Printf("Unable to retrieve data from sheet: %v", err)
			return nil, nil, err
		}
		// google sheet read, then parse image list func
		parseImgList, parseExImgList := parseImageList(resp)

		imageList = append(imageList, parseImgList...)
		exceptImageList = append(exceptImageList, parseExImgList...)
	}
	return imageList, exceptImageList, nil
}

// Add a new sheet tab in target google sheet
func (gsheet *Gsheet) AddNewSheet(newSheetTitle string) error {

	spreadsheetId := gsheet.SpreadsheetId
	srv := gsheet.Service

	reqAddSheet := sheets.Request{
		AddSheet: &sheets.AddSheetRequest{
			Properties: &sheets.SheetProperties{
				Title: newSheetTitle,
			},
		},
	}

	rb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{&reqAddSheet},
	}

	resp, err := srv.Spreadsheets.BatchUpdate(spreadsheetId, rb).Context(gsheet.Ctx).Do()
	if err != nil {
		log.Error.Printf("Can`t add a new sheet: %v", err)
		return err
	}
	log.Info.Println(resp)

	return nil

}

// Write to release image list info in target sheet tab
func (gsheet *Gsheet) SetGsheet(imageList []string) error {

	// Instance information set
	spreadsheetId := gsheet.SpreadsheetId
	writeRange := gsheet.WriteRange // "*.tar.gz!B2"
	srv := gsheet.Service

	values := make([][]interface{}, len(imageList))
	for idx, v := range imageList {
		values[idx] = append(values[idx], idx+1)
		values[idx] = append(values[idx], v)
	}

	rb := &sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
	}
	rb.Data = append(rb.Data, &sheets.ValueRange{
		Range:  writeRange,
		Values: values,
	})

	resp, err := srv.Spreadsheets.Values.BatchUpdate(spreadsheetId, rb).Context(gsheet.Ctx).Do()

	if err != nil {
		log.Error.Printf("Unable to append data to sheet: %v", err)
		return err
	}
	log.Info.Println(resp)

	return nil
}

// Parse two image lists(export option is true or false)
func parseImageList(resp *sheets.ValueRange) (imageList []string, exceptImageList []string) {

	if len(resp.Values) == 0 {
		log.Info.Println("No data found.")
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
	return imageList, exceptImageList
}
