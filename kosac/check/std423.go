package check

import (
	_ "embed"
	"regexp"
	"strings"
)

// School codes from sheet 03 of the 20261105-v2.3 survey vocabulary.
//
//go:embed neis_20261105.txt
var schoolCodesText423 string
var schoolCodes423 = strings.Fields(schoolCodesText423)

// std423Item is a row of the foundation's survey standard code table
// (표준코드표(설문용) 02, codeListVersionID 20261105-v2.3). Only survey
// items are listed; business-record-only codes are not, so S10 rejects them.
type std423Item struct {
	ID, Name, Level, Type string
	Sets                  []string       // internal code sets; empty for external or numeric items
	Pattern               *regexp.Regexp // format of an external code, when the table gives one
	Min, Max              int            // numeric range of valid codes, when the table gives one
}

type std423Code struct{ Code, Label, Start, End string }

var std423Items = map[string]std423Item{
	"PERSON.GENDER":  {ID: "PERSON.GENDER", Name: "성별", Level: "명목", Type: "문자", Sets: []string{"GENDER"}},
	"PERSON.ROLE":    {ID: "PERSON.ROLE", Name: "참여자 구분", Level: "명목", Type: "문자", Sets: []string{"ROLE"}},
	"SCHOOL.ID":      {ID: "SCHOOL.ID", Name: "학교 코드", Level: "명목", Type: "문자"},
	"SCHOOL.GRADE":   {ID: "SCHOOL.GRADE", Name: "학년", Level: "서열", Type: "숫자", Min: 1, Max: 6},
	"REGION.SIDO":    {ID: "REGION.SIDO", Name: "시도", Level: "명목", Type: "문자", Sets: []string{"SIDO"}},
	"REGION.SIGUNGU": {ID: "REGION.SIGUNGU", Name: "시군구", Level: "명목", Type: "문자", Pattern: regexp.MustCompile(`^[0-9]{5}$`)},
	"SCHOOL.LEVEL":   {ID: "SCHOOL.LEVEL", Name: "학교급", Level: "명목", Type: "문자", Sets: []string{"SCHOOL_LEVEL"}},
	"PERSON.AGE":     {ID: "PERSON.AGE", Name: "만 나이", Level: "비율", Type: "숫자"},
	"PERSON.AGE_GRP": {ID: "PERSON.AGE_GRP", Name: "연령대", Level: "서열", Type: "문자", Sets: []string{"AGE_GRP"}},
	"PERSON.CARE":    {ID: "PERSON.CARE", Name: "대상 유형", Level: "명목", Type: "문자", Sets: []string{"CARE_GRP", "CARE"}},
	"SCHOOL.OFCDC":   {ID: "SCHOOL.OFCDC", Name: "시도교육청", Level: "명목", Type: "문자", Sets: []string{"OFCDC"}},
	"SCHOOL.KEDI_ID": {ID: "SCHOOL.KEDI_ID", Name: "KEDI 학교코드", Level: "명목", Type: "문자", Pattern: regexp.MustCompile(`^[0-9]{9}$`)},
	"REGION.SCALE":   {ID: "REGION.SCALE", Name: "지역 규모", Level: "명목", Type: "문자", Sets: []string{"SCALE"}},
	"PERSON.JOB":     {ID: "PERSON.JOB", Name: "직업", Level: "명목", Type: "문자", Sets: []string{"JOB"}},
	"PERSON.EDU":     {ID: "PERSON.EDU", Name: "최종 학력", Level: "서열", Type: "문자", Sets: []string{"EDU"}},
	"PERSON.MAJOR":   {ID: "PERSON.MAJOR", Name: "전공 계열", Level: "명목", Type: "문자", Sets: []string{"MAJOR"}},
	"SURVEY.WAVE":    {ID: "SURVEY.WAVE", Name: "조사 시점", Level: "명목", Type: "문자", Sets: []string{"WAVE"}},
	"SCHOOL.SIDO":    {ID: "SCHOOL.SIDO", Name: "학교 소재 시도", Level: "명목", Type: "문자", Sets: []string{"SIDO"}},
	"SCHOOL.SIGUNGU": {ID: "SCHOOL.SIGUNGU", Name: "학교 소재 시군구", Level: "명목", Type: "문자", Pattern: regexp.MustCompile(`^[0-9]{5}$`)},
}

func codes423(pairs ...string) []std423Code {
	var out []std423Code
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, std423Code{Code: pairs[i], Label: pairs[i+1]})
	}
	return out
}

var std423Sets = map[string][]std423Code{
	"GENDER":       codes423("1", "남", "2", "여", "9", "미상", "3", "기타", "-7", "응답하지 않음"),
	"AGE_GRP":      codes423("01", "18세 이하", "02", "19~29세", "03", "30~39세", "04", "40~49세", "05", "50~59세", "06", "60~64세", "07", "65세 이상"),
	"ROLE":         codes423("01", "유아", "02", "초등", "03", "중등", "04", "고등", "05", "학교밖청소년", "06", "대학생", "07", "성인", "08", "노인", "09", "교원", "10", "기타", "11", "미상"),
	"SCHOOL_LEVEL": codes423("01", "유치원", "02", "초등학교", "03", "중학교", "04", "고등학교", "05", "특수학교", "06", "대학", "07", "대학원", "99", "기타"),
	"SIDO": {
		{"11", "서울특별시", "", ""}, {"26", "부산광역시", "", ""}, {"27", "대구광역시", "", ""}, {"28", "인천광역시", "", ""},
		{"29", "광주광역시", "", "2026-06-30"}, {"30", "대전광역시", "", ""}, {"31", "울산광역시", "", ""}, {"36", "세종특별자치시", "", ""},
		{"41", "경기도", "", ""}, {"42", "강원도", "", "2023-06-10"}, {"43", "충청북도", "", ""}, {"44", "충청남도", "", ""},
		{"45", "전라북도", "", "2024-01-17"}, {"46", "전라남도", "", "2026-06-30"}, {"47", "경상북도", "", ""}, {"48", "경상남도", "", ""},
		{"50", "제주특별자치도", "", ""}, {"51", "강원특별자치도", "2023-06-11", ""}, {"52", "전북특별자치도", "2024-01-18", ""},
		{"12", "전남광주통합특별시", "2026-07-01", ""},
	},
	"OFCDC": codes423("B10", "서울특별시교육청", "C10", "부산광역시교육청", "D10", "대구광역시교육청", "E10", "인천광역시교육청",
		"F10", "광주광역시교육청", "G10", "대전광역시교육청", "H10", "울산광역시교육청", "I10", "세종특별자치시교육청",
		"J10", "경기도교육청", "K10", "강원특별자치도교육청", "M10", "충청북도교육청", "N10", "충청남도교육청",
		"P10", "전북특별자치도교육청", "Q10", "전라남도교육청", "R10", "경상북도교육청", "S10", "경상남도교육청", "T10", "제주특별자치도교육청"),
	"CARE_GRP": codes423("G", "일반", "C", "사회적 배려대상", "O", "학교 밖 청소년"),
	"CARE": codes423("ECO", "경제적(기초수급·차상위)", "GEO", "지리적(읍면·도서벽지)", "CUL", "사회문화적(다문화·북한이탈)",
		"FAM", "가정환경적(한부모·조손·다자녀)", "PHY", "신체적(특수교육·장애)", "MER", "국가유공자", "ETC", "기타"),
	"SCALE": codes423("A", "특별·광역시", "B", "시", "C", "읍", "D", "면", "E", "도서벽지"),
	"JOB": codes423("1", "관리자", "2", "전문가 및 관련 종사자", "3", "사무 종사자", "4", "서비스 종사자", "5", "판매 종사자",
		"6", "농림어업 숙련 종사자", "7", "기능원 및 관련 기능 종사자", "8", "장치·기계 조작 및 조립 종사자", "9", "단순노무 종사자",
		"10", "군인", "21", "학생", "22", "전업주부", "23", "무직", "29", "기타"),
	"EDU":   codes423("1", "중졸 이하", "2", "고졸", "3", "전문대졸", "4", "대졸", "5", "대학원 이상"),
	"MAJOR": codes423("1", "인문", "2", "사회", "3", "교육", "4", "공학", "5", "자연", "6", "의약", "7", "예체능"),
	"WAVE":  codes423("1", "사전", "2", "사후"),
}
