package kind

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

const sampleHTML = `<!DOCTYPE html>
<html><body><table><tbody>
<tr><td>삼성전자</td><td>코스피</td><td>005930</td><td>전기·전자</td><td>반도체</td><td>1975-06-11</td></tr>
<tr><td>SK하이닉스</td><td>코스피</td><td>000660</td><td>전기·전자</td><td>메모리</td><td>1996-12-26</td></tr>
<tr><td>잘못된행</td><td>코스피</td><td>NOT_A_CODE</td></tr>
</tbody></table></body></html>`

// eucKRBytes encodes the UTF-8 string to EUC-KR bytes, matching what KIND actually serves.
func eucKRBytes(t *testing.T, s string) []byte {
	t.Helper()
	enc := korean.EUCKR.NewEncoder()
	out, _, err := transform.Bytes(enc, []byte(s))
	require.NoError(t, err, "EUC-KR 인코딩 실패")
	return out
}

func TestFetchInstruments_ParsesHTML(t *testing.T) {
	body := eucKRBytes(t, sampleHTML)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "marketType=stockMkt")
		w.Header().Set("Content-Type", "text/html; charset=euc-kr")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	inst, err := c.FetchInstruments(context.Background(), "KOSPI")
	require.NoError(t, err)
	require.Equal(t, 2, len(inst), "잘못된 행은 필터링되어야 함")
	assert.Equal(t, "005930", inst[0].Symbol)
	assert.Equal(t, "삼성전자", inst[0].Name)
	assert.Equal(t, "KOSPI", inst[0].Exchange)
	assert.Equal(t, "KRW", inst[0].Currency)
}

func TestFetchInstruments_KOSDAQ(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Contains(t, r.URL.RawQuery, "marketType=kosdaqMkt")
		_, _ = w.Write([]byte(`<html><body><table></table></body></html>`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, _ = c.FetchInstruments(context.Background(), "KOSDAQ")
	assert.True(t, called)
}
