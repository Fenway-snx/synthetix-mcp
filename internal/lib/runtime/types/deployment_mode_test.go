package types

import (
	"testing"
)

func Test_DeploymentMode(t *testing.T) {
	t.Run("all constants have distinct values", func(t *testing.T) {
		modes := []struct {
			name string
			val  DeploymentMode
		}{
			{"Unknown", DeploymentMode_Unknown},
			{"Local", DeploymentMode_Local},
			{"Development", DeploymentMode_Development},
			{"Staging", DeploymentMode_Staging},
			{"Production", DeploymentMode_Production},
			{"Custom", DeploymentMode_Custom},
		}

		seen := make(map[DeploymentMode]string, len(modes))
		for _, m := range modes {
			if prev, ok := seen[m.val]; ok {
				t.Errorf("%s (%d) collides with %s", m.name, m.val, prev)
			}
			seen[m.val] = m.name
		}
	})

	t.Run("integer values", func(t *testing.T) {
		cases := []struct {
			name string
			got  DeploymentMode
			want DeploymentMode
		}{
			{"Unknown", DeploymentMode_Unknown, 0},
			{"Local", DeploymentMode_Local, 1},
			{"Development", DeploymentMode_Development, 2},
			{"Staging", DeploymentMode_Staging, 4},
			{"Production", DeploymentMode_Production, 8},
			{"Custom", DeploymentMode_Custom, 16},
		}

		for _, tc := range cases {
			if tc.got != tc.want {
				t.Errorf("DeploymentMode_%s = %d, want %d", tc.name, tc.got, tc.want)
			}
		}
	})
}

func Test_DeploymentMode_CanonicalString(t *testing.T) {
	tests := []struct {
		mode DeploymentMode
		want string
	}{
		{DeploymentMode_Unknown, ""},
		{DeploymentMode_Local, "local"},
		{DeploymentMode_Development, "development"},
		{DeploymentMode_Staging, "staging"},
		{DeploymentMode_Production, "production"},
		{DeploymentMode_Custom, "custom"},
		{DeploymentMode(999), ""},
	}

	for _, tc := range tests {
		got := tc.mode.CanonicalString()
		if got != tc.want {
			t.Errorf("DeploymentMode(%d).CanonicalString() = %q, want %q",
				tc.mode, got, tc.want)
		}
	}
}

func Test_ParseDeploymentMode(t *testing.T) {
	tests := []struct {
		input string
		want  DeploymentMode
	}{
		// Canonical lowercase
		{"custom", DeploymentMode_Custom},
		{"dev", DeploymentMode_Development},
		{"development", DeploymentMode_Development},
		{"local", DeploymentMode_Local},
		{"prod", DeploymentMode_Production},
		{"production", DeploymentMode_Production},
		{"staging", DeploymentMode_Staging},

		// Mixed case
		{"Custom", DeploymentMode_Custom},
		{"DEV", DeploymentMode_Development},
		{"Development", DeploymentMode_Development},
		{"LOCAL", DeploymentMode_Local},
		{"Prod", DeploymentMode_Production},
		{"PRODUCTION", DeploymentMode_Production},
		{"Staging", DeploymentMode_Staging},

		// Surrounding whitespace
		{"  local  ", DeploymentMode_Local},
		{"\tprod\n", DeploymentMode_Production},

		// Empty / whitespace → Unknown
		{"", DeploymentMode_Unknown},
		{"  ", DeploymentMode_Unknown},

		// Unrecognised → Custom
		{"unknown", DeploymentMode_Custom},
		{"bogus", DeploymentMode_Custom},
		{"localx", DeploymentMode_Custom},
	}

	for _, tc := range tests {
		got, _ := ParseDeploymentMode(tc.input)
		if got != tc.want {
			t.Errorf("ParseDeploymentMode(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func Test_ParseDeploymentMode_Slice(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  DeploymentMode
	}{
		{"nil slice", nil, DeploymentMode_Unknown},
		{"empty slice", []string{}, DeploymentMode_Unknown},
		{"single element", []string{"local"}, DeploymentMode_Local},
		{"first non-empty wins", []string{"staging", "production"}, DeploymentMode_Staging},
		{"skips leading empty", []string{"", "  ", "prod"}, DeploymentMode_Production},
		{"all empty", []string{"", "  ", "\t"}, DeploymentMode_Unknown},
		{"first non-empty is unrecognised", []string{"", "myenv", "local"}, DeploymentMode_Custom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := ParseDeploymentMode(tc.input)
			if got != tc.want {
				t.Errorf("ParseDeploymentMode(%v) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}
