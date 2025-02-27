package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wandb/wsm/pkg/deployer"
	"github.com/wandb/wsm/pkg/helm"
	"github.com/wandb/wsm/pkg/images"
	"github.com/wandb/wsm/pkg/term/pkgm"
	"github.com/wandb/wsm/pkg/utils"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(DownloadCmd())
}

func downloadChartImages(
	url string,
	name string,
	version string,
	vals map[string]interface{},
) ([]string, error) {
	chartsDir := "bundle/charts"
	if err := os.MkdirAll(chartsDir, 0755); err != nil {
		return nil, err
	}

	chart, err := helm.DownloadChart(
		url,
		name,
		version,
		chartsDir,
	)
	if err != nil {
		return nil, err
	}

	runs, err := helm.GetRuntimeObjects(chart, vals)
	if err != nil {
		return nil, err
	}
	return helm.ExtractImages(runs), nil
}

func DownloadCmd() *cobra.Command {
	var platform string

	cmd := &cobra.Command{
		Use: "download",
		Run: func(cmd *cobra.Command, args []string) {
			_ = os.RemoveAll("bundle")
			// Fetch the latest tag for the controller
			operatorTag, err := getMostRecentTag("wandb/controller")
			if err != nil {
				fmt.Printf("Error fetching the latest operator-wandb controller tag: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Downloading operator helm chart")
			operatorImgs, _ := downloadChartImages(
				helm.WandbHelmRepoURL,
				helm.WandbOperatorChart,
				"", // empty version means latest
				map[string]interface{}{
					"image": map[string]interface{}{
						"tag": operatorTag,
					},
				},
			)

			spec, err := deployer.GetChannelSpec("")
			if err != nil {
				panic(err)
			}
			// Create a copy of the spec to download additional images without writing changes to the filesystem
			dlSpec, err := deployer.GetChannelSpec("")
			if err != nil {
				panic(err)
			}

			// Enable weave-trace in the chart values
			if dlWeaveTrace, ok := dlSpec.Values["weave-trace"]; ok {
				dlWeaveTrace.(map[string]interface{})["install"] = true
			}

			fmt.Println("Downloading wandb helm chart")
			wandbImgs, _ := downloadChartImages(
				spec.Chart.URL,
				spec.Chart.Name,
				spec.Chart.Version,
				dlSpec.Values,
			)

			imgs := utils.RemoveDuplicates(append(wandbImgs, operatorImgs...))
			if len(imgs) == 0 {
				fmt.Println("No images to download.")
				os.Exit(1)
			}

			data := make(map[string]interface{})
			data["wandb"] = spec.Values
			yamlData, err := yaml.Marshal(data)
			if err != nil {
				panic(err)
			}
			if err = os.WriteFile("bundle/spec.yaml", yamlData, 0644); err != nil {
				panic(err)
			}

			cb := func(pkg string) {
				path := "bundle/images/" + pkg
				_ = os.MkdirAll(path, 0755)
				err := images.Download(pkg, path+"/image.tgz", platform)
				if err != nil {
					fmt.Println(err)
				}
			}

			if _, err := pkgm.New(imgs, cb).Run(); err != nil {
				fmt.Println("Error deploying:", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&platform, "platform", "p", "linux/amd64", "Platform to download images for")

	return cmd
}
