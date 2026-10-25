package tmdb

import (
	"reflect"
	"testing"
)

func TestRussianTitleVariant(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"Semejka Addams", "Семейка Аддамс", true},
		{"Nemyslimoe", "Немыслимое", true},
		{"Skorostj", "Скорость", true},
		{"Uzhastiki Ozhivshie Risunki", "Ужастики Ожившие Рисунки", true},
		{"Ivan Carevich i Seryj Volk", "Иван Царевич и Серый Волк", true},
		{"Wednesday", "", false},
		{"The Addams Family", "", false},
	}
	for _, test := range tests {
		got, ok := russianTitleVariant(test.input)
		if got != test.want || ok != test.ok {
			t.Errorf("russianTitleVariant(%q) = %q, %v; want %q, %v", test.input, got, ok, test.want, test.ok)
		}
	}
}

func TestLookupTitleVariantsCleansFilmCount(t *testing.T) {
	want := []string{"Semejka Addams (2 Filma)", "Semejka Addams", "Семейка Аддамс", "Семейка Аддамс (2 Филма)"}
	if got := lookupTitleVariants("Semejka Addams (2 Filma)"); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v; want %#v", got, want)
	}
}
