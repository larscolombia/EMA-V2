package openai

import "testing"

func TestSanitizeEnv(t *testing.T){
  cases := map[string]string{
    `"sk-abc"`: "sk-abc",
    `'sk-abc'`: "sk-abc",
    " sk-xyz ": "sk-xyz",
    "sk-no-quotes": "sk-no-quotes",
    "\"incomplete": "\"incomplete", // no recorta si no hay cierre
  }
  for in, exp := range cases {
    got := sanitizeEnv(in)
    if got != exp { t.Errorf("sanitizeEnv(%q)=%q; want %q", in, got, exp) }
  }
}
