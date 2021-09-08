## Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

    helm repo add coc-repo https://gympass.github.io/cdn-origin-controller/

If you had already added this repo earlier, run `helm repo update` to retrieve
the latest versions of the packages.  You can then run `helm search repo
coc-repo` to see the charts.

To install the cdn-origin-controller chart:

    helm install my-cdn-origin-controller coc-repo/cdn-origin-controller

To uninstall the chart:

    helm delete my-cdn-origin-controller
