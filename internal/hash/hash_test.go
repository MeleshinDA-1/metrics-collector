package hash

import "testing"

const testKey = "supersecret"

func TestSignIsStableAndKeyDependent(t *testing.T) {
	data := []byte(`[{"id":"Alloc","type":"gauge","value":42}]`)

	signature := Sign(data, testKey)
	if signature != Sign(data, testKey) {
		t.Fatal("Sign returned different signatures for the same input")
	}
	if len(signature) != 44 {
		t.Fatalf("signature length = %d, want %d", len(signature), 44)
	}
	if signature == Sign(data, "another key") {
		t.Fatal("signatures computed with different keys are equal")
	}
	if signature == Sign([]byte("other data"), testKey) {
		t.Fatal("signatures computed for different data are equal")
	}
}

func TestEqual(t *testing.T) {
	data := []byte("metrics batch")

	tests := []struct {
		name      string
		signature string
		want      bool
	}{
		{
			name:      "matching signature",
			signature: Sign(data, testKey),
			want:      true,
		},
		{
			name:      "signature of other data",
			signature: Sign([]byte("other batch"), testKey),
			want:      false,
		},
		{
			name:      "signature made with another key",
			signature: Sign(data, "another key"),
			want:      false,
		},
		{
			name:      "not a base64 string",
			signature: "not a signature",
			want:      false,
		},
		{
			name:      "empty signature",
			signature: "",
			want:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Equal(data, testKey, test.signature); got != test.want {
				t.Fatalf("Equal = %v, want %v", got, test.want)
			}
		})
	}
}
