package service_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"gopkg.in/yaml.v2"

	"github.com/google/go-cmp/cmp"

	"github.com/Bocmah/phpdocker-gen/pkg/service"
)

// Helpers
func yamlMarshal(t *testing.T, source interface{}) []byte {
	t.Helper()

	res, err := yaml.Marshal(source)

	if err != nil {
		t.Fatalf("failed to marshal: %s", err)
	}

	return res
}

func createTmpFile(t *testing.T, fs afero.Fs, pattern string) afero.File {
	t.Helper()

	tmpfile, err := afero.TempFile(fs, "", pattern)

	if err != nil {
		t.Fatalf("failed to create tempfile: %s", err)
	}

	return tmpfile
}

func writeToTmpFile(t *testing.T, tmpfile afero.File, content []byte) {
	t.Helper()

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatalf("failed to write to tempfile: %s", err)
	}
}

func closeTmpFile(t *testing.T, tmpfile afero.File) {
	t.Helper()

	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close tempfile: %s", err)
	}
}

func TestLoadConfigFromFileIncorrectPath(t *testing.T) {
	_, err := service.LoadConfigFromFile("/incorrect")

	if err == nil {
		t.Fatalf("encountered nil err when loading config from incorrect path")
	}

	if !strings.Contains(err.Error(), "read config") {
		t.Fatalf("incorrect err value: %s", err.Error())
	}
}

func TestLoadConfigFromIncorrectFile(t *testing.T) {
	content := []byte("some random string")

	service.AppFs = afero.NewMemMapFs()

	tmpfile := createTmpFile(t, service.AppFs, "example")

	writeToTmpFile(t, tmpfile, content)
	closeTmpFile(t, tmpfile)

	_, loadConfigErr := service.LoadConfigFromFile(tmpfile.Name())

	if loadConfigErr == nil {
		t.Fatalf("encountered nil err when loading config from incorrect file")
	}

	if !strings.Contains(loadConfigErr.Error(), "parse config") {
		t.Fatalf("incorrect err value: %s", loadConfigErr.Error())
	}
}

func TestLoadConfigFromFileFailedValidation(t *testing.T) {
	testConf := map[string]interface{}{
		"appName":     "phpdocker-gen",
		"projectRoot": "/home/user/projects/test",
		"outputPath":  "/home/user/output",
		"services": map[interface{}]interface{}{
			"php": map[interface{}]interface{}{
				"version": "7.4",
				"extensions": []interface{}{
					"mbstring",
					"zip",
					"exif",
					"pcntl",
					"gd",
				},
			},
			// Incorrect database system
			"database": map[interface{}]interface{}{
				"system":       "mysqlll",
				"version":      "5.7",
				"name":         "test-db",
				"port":         3306,
				"username":     "bocmah",
				"password":     "test",
				"rootPassword": "testRoot",
			},
		},
	}

	yamlTestConf := yamlMarshal(t, testConf)

	service.AppFs = afero.NewMemMapFs()

	tmpfile := createTmpFile(t, service.AppFs, "*.yaml")

	writeToTmpFile(t, tmpfile, yamlTestConf)
	closeTmpFile(t, tmpfile)

	_, err := service.LoadConfigFromFile(tmpfile.Name())

	if err == nil {
		t.Fatalf("encountered nil err when loading config with failed validation")
	}

	if _, ok := err.(*service.ValidationErrors); !ok {
		t.Fatalf("incorrect err value %v", err)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	testConf := map[string]interface{}{
		"appName":     "phpdocker-gen",
		"projectRoot": "/home/user/projects/test",
		"outputPath":  "/home/user/output",
		"services": map[interface{}]interface{}{
			"php": map[interface{}]interface{}{
				"version": "7.4",
				"extensions": []interface{}{
					"mbstring",
					"zip",
					"exif",
					"pcntl",
					"gd",
				},
			},
			"nginx": map[interface{}]interface{}{
				"httpPort":   80,
				"serverName": "test-server",
				"fastCGI": map[interface{}]interface{}{
					"passPort":           9000,
					"readTimeoutSeconds": 60,
				},
			},
			"nodejs": map[interface{}]interface{}{
				"version": "10",
			},
			"database": map[interface{}]interface{}{
				"system":       "mysql",
				"version":      "5.7",
				"name":         "test-db",
				"port":         3306,
				"username":     "bocmah",
				"password":     "test",
				"rootPassword": "testRoot",
			},
		},
	}

	yamlTestConf := yamlMarshal(t, testConf)

	service.AppFs = afero.NewMemMapFs()

	tmpfile := createTmpFile(t, service.AppFs, "*.yaml")

	writeToTmpFile(t, tmpfile, yamlTestConf)
	closeTmpFile(t, tmpfile)

	got, loadErr := service.LoadConfigFromFile(tmpfile.Name())

	if loadErr != nil {
		t.Errorf("Got error when loading correct config. Error - %v, Value - %v", loadErr, got)
		return
	}

	want := &service.FullConfig{
		AppName:     "phpdocker-gen",
		ProjectRoot: "/home/user/projects/test",
		OutputPath:  "/home/user/output",
		Services: &service.ServicesConfig{
			PHP: &service.PHPConfig{
				Version:    "7.4",
				Extensions: []string{"mbstring", "zip", "exif", "pcntl", "gd", "pdo_mysql"},
			},
			Nginx: &service.NginxConfig{
				HTTPPort:   80,
				HTTPSPort:  443,
				ServerName: "test-server",
				FastCGI: &service.FastCGI{
					PassPort:           9000,
					ReadTimeoutSeconds: 60,
				},
			},
			NodeJS: &service.NodeJSConfig{
				Version: "10",
			},
			Database: &service.DatabaseConfig{
				System:  service.MySQL,
				Version: "5.7",
				Name:    "test-db",
				Port:    3306,
				Credentials: service.Credentials{
					Username:     "bocmah",
					Password:     "test",
					RootPassword: "testRoot",
				},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("incorrectly loaded configuration (-want +got):\n%s", diff)
	}
}

func TestLoadConfigFromFile_OneService(t *testing.T) {
	testConf := map[string]interface{}{
		"appName":     "phpdocker-gen",
		"projectRoot": "/home/user/projects/test",
		"outputPath":  "/home/user/output",
		"services": map[interface{}]interface{}{
			"php": map[interface{}]interface{}{
				"version": "7.4",
				"extensions": []interface{}{
					"mbstring",
					"zip",
					"exif",
					"pcntl",
					"gd",
				},
			},
		},
	}

	yamlTestConf := yamlMarshal(t, testConf)

	service.AppFs = afero.NewMemMapFs()

	tmpfile := createTmpFile(t, service.AppFs, "*.yaml")

	writeToTmpFile(t, tmpfile, yamlTestConf)
	closeTmpFile(t, tmpfile)

	got, loadErr := service.LoadConfigFromFile(tmpfile.Name())

	if loadErr != nil {
		t.Errorf("Got error when loading correct config. Error - %v, Value - %v", loadErr, got)
		return
	}

	want := &service.FullConfig{
		AppName:     "phpdocker-gen",
		ProjectRoot: "/home/user/projects/test",
		OutputPath:  "/home/user/output",
		Services: &service.ServicesConfig{
			PHP: &service.PHPConfig{
				Version:    "7.4",
				Extensions: []string{"mbstring", "zip", "exif", "pcntl", "gd"},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("incorrectly loaded configuration (-want +got):\n%s", diff)
	}
}

func TestFullConfigValid_Validate(t *testing.T) {
	conf := &service.FullConfig{
		AppName:     "phpdocker-gen",
		ProjectRoot: "/home/user/projects/test",
		OutputPath:  "/home/user/output",
		Services: &service.ServicesConfig{
			PHP: &service.PHPConfig{
				Version:    "7.4",
				Extensions: []string{"mbstring", "zip", "exif", "pcntl", "gd", "pdo_mysql"},
			},
			Nginx: &service.NginxConfig{
				HTTPPort:   80,
				HTTPSPort:  443,
				ServerName: "test-server",
				FastCGI: &service.FastCGI{
					PassPort:           9000,
					ReadTimeoutSeconds: 60,
				},
			},
			NodeJS: &service.NodeJSConfig{
				Version: "10",
			},
			Database: &service.DatabaseConfig{
				System:  service.MySQL,
				Version: "5.7",
				Name:    "test-db",
				Port:    3306,
				Credentials: service.Credentials{
					Username:     "bocmah",
					Password:     "test",
					RootPassword: "testRoot",
				},
			},
		},
	}

	validationErr := conf.Validate()

	if validationErr != nil {
		t.Fatalf("Encountered non-nil validation error on valid config: %s", validationErr)
	}
}

func TestFullConfigInvalid_Validate(t *testing.T) {
	tests := map[string]struct {
		conf         *service.FullConfig
		expectedErrs []string
	}{
		"without app name and project root": {
			conf: &service.FullConfig{
				OutputPath: "/home/user/output",
				Services: &service.ServicesConfig{
					PHP: &service.PHPConfig{
						Version:    "7.4",
						Extensions: []string{"mbstring", "zip", "exif", "pcntl", "gd", "pdo_mysql"},
					},
					Nginx: &service.NginxConfig{
						HTTPPort:   80,
						HTTPSPort:  443,
						ServerName: "test-server",
						FastCGI: &service.FastCGI{
							PassPort:           9000,
							ReadTimeoutSeconds: 60,
						},
					},
					NodeJS: &service.NodeJSConfig{
						Version: "10",
					},
					Database: &service.DatabaseConfig{
						System:  service.MySQL,
						Version: "5.7",
						Name:    "test-db",
						Port:    3306,
						Credentials: service.Credentials{
							Username:     "bocmah",
							Password:     "test",
							RootPassword: "testRoot",
						},
					},
				},
			},
			expectedErrs: []string{"App name is required", "Project root is required"},
		},
		"Services is set but it is empty": {
			conf: &service.FullConfig{
				AppName:     "phpdocker-gen",
				ProjectRoot: "/home/user/projects/test",
				OutputPath:  "/home/user/output",
				Services:    &service.ServicesConfig{},
			},
			expectedErrs: []string{"At least one service is required"},
		},
		"Services is not set": {
			conf: &service.FullConfig{
				AppName:     "phpdocker-gen",
				ProjectRoot: "/home/user/projects/test",
				OutputPath:  "/home/user/output",
			},
			expectedErrs: []string{"At least one service is required"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			validationErr := tc.conf.Validate()

			if validationErr == nil {
				t.Fatalf("Encountered nil validation error on invalid config: %s", validationErr)
			}

			stringErr := validationErr.Error()

			for _, expectedErr := range tc.expectedErrs {
				if !strings.Contains(stringErr, expectedErr) {
					t.Fatalf("validation err %s does not contain expected err %s", stringErr, expectedErr)
				}
			}
		})
	}
}

func TestFullConfig_GetServiceFiles(t *testing.T) {
	outputPath := "/home/user/output"

	conf := &service.FullConfig{
		AppName:     "phpdocker-gen",
		ProjectRoot: "/home/user/projects/test",
		OutputPath:  outputPath,
		Services: &service.ServicesConfig{
			PHP: &service.PHPConfig{
				Version:    "7.4",
				Extensions: []string{"mbstring", "zip", "exif", "pcntl", "gd", "pdo_mysql"},
			},
			Nginx: &service.NginxConfig{
				HTTPPort:   80,
				HTTPSPort:  443,
				ServerName: "test-server",
				FastCGI: &service.FastCGI{
					PassPort:           9000,
					ReadTimeoutSeconds: 60,
				},
			},
			NodeJS: &service.NodeJSConfig{
				Version: "10",
			},
			Database: &service.DatabaseConfig{
				System:  service.MySQL,
				Version: "5.7",
				Name:    "test-db",
				Port:    3306,
				Credentials: service.Credentials{
					Username:     "bocmah",
					Password:     "test",
					RootPassword: "testRoot",
				},
			},
		},
	}

	want := service.Files{
		service.PHP: []*service.File{
			{
				Type:         service.Dockerfile,
				PathOnHost:   filepath.Join(outputPath, "php/Dockerfile"),
				TemplatePath: "/php/php.dockerfile.gotmpl",
			},
		},
		service.Nginx: []*service.File{
			{
				Type:            service.ConfigFile,
				PathOnHost:      filepath.Join(outputPath, "nginx/conf.d/app.conf"),
				PathInContainer: "/etc/nginx/conf.d/app.conf",
				TemplatePath:    "/nginx/conf.gotmpl",
			},
		},
		service.NodeJS: []*service.File{
			{
				Type:         service.Dockerfile,
				PathOnHost:   filepath.Join(outputPath, "nodejs/Dockerfile"),
				TemplatePath: "/nodejs/nodejs.dockerfile.gotmpl",
			},
		},
	}

	got := conf.GetServiceFiles()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("conf.GetServiceFiles() mismatch (-want +got):\n%s", diff)
	}
}

func TestFullConfig_GetEnvironment(t *testing.T) {
	conf := &service.FullConfig{
		AppName:     "phpdocker-gen",
		ProjectRoot: "/home/user/projects/test",
		OutputPath:  "/home/user/output",
		Services: &service.ServicesConfig{
			PHP: &service.PHPConfig{
				Version:    "7.4",
				Extensions: []string{"mbstring", "zip", "exif", "pcntl", "gd", "pdo_mysql"},
			},
			Nginx: &service.NginxConfig{
				HTTPPort:   80,
				HTTPSPort:  443,
				ServerName: "test-server",
				FastCGI: &service.FastCGI{
					PassPort:           9000,
					ReadTimeoutSeconds: 60,
				},
			},
			NodeJS: &service.NodeJSConfig{
				Version: "10",
			},
		},
	}

	env := conf.GetEnvironment()

	if env != nil {
		t.Errorf("encountered non-nil environment when nil is expected: %s", env)
	}

	conf.Services.Database = &service.DatabaseConfig{
		System:  service.MySQL,
		Version: "5.7",
		Name:    "test-db",
		Port:    3306,
		Credentials: service.Credentials{
			Username:     "bocmah",
			Password:     "test",
			RootPassword: "testRoot",
		},
	}

	env = conf.GetEnvironment()

	wantEnv := service.Environment{
		service.Database: {
			"MYSQL_ROOT_PASSWORD": "testRoot",
			"MYSQL_DATABASE":      "test-db",
			"MYSQL_USER":          "bocmah",
			"MYSQL_PASSWORD":      "test",
		},
	}

	if diff := cmp.Diff(wantEnv, env); diff != "" {
		t.Errorf("conf.GetEnvironment() mismatch (-want +got):\n%s", diff)
	}
}

func TestFullConfig_GetOutputPath(t *testing.T) {
	conf := &service.FullConfig{
		AppName:     "phpdocker-gen",
		ProjectRoot: "/home/user/projects/test",
		Services: &service.ServicesConfig{
			PHP: &service.PHPConfig{
				Version:    "7.4",
				Extensions: []string{"mbstring", "zip", "exif", "pcntl", "gd", "pdo_mysql"},
			},
			Nginx: &service.NginxConfig{
				HTTPPort:   80,
				HTTPSPort:  443,
				ServerName: "test-server",
				FastCGI: &service.FastCGI{
					PassPort:           9000,
					ReadTimeoutSeconds: 60,
				},
			},
			NodeJS: &service.NodeJSConfig{
				Version: "10",
			},
		},
	}

	want := filepath.Join(conf.ProjectRoot, ".docker")
	got := conf.GetOutputPath()

	if got != want {
		t.Errorf("incorrect output path for config without explicitly set OutputPath. got %s, want %s", got, want)
	}

	conf.OutputPath = "/home/test/output"

	want = "/home/test/output"
	got = conf.GetOutputPath()

	if got != want {
		t.Errorf("incorrect output path for config with explicitly set OutputPath. got %s, want %s", got, want)
	}
}
