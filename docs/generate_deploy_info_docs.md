# Deploy Info JSON

## The `deploy_info.json` file in the `vSwarm_deploy_metadata.tar.gz`  is used to identify the relative file paths for the Knative YAML manifests for deploying vSwarm functions. It is generated using `workloads/container/generate_deploy_info.py` Python script, which outputs a JSON that embeds the yaml-location and pre-deployment commands for every vSwarm function. It also contains the path of YAML files needed as part of the pre-deployment commands to run certain vSwarm benchmarks, for example the `online-shop-database` which requires to be deployed before running `cartservice` benchmark.

While the `deploy_info.json` file ships with the `vSwarm_deploy_metadata.tar.gz`, In order to regenerate the `deploy_info.json` run from the root of this repository:
```console
tar -xzvf workloads/container/vSwarm_deploy_metadata.tar.gz -C workloads/container
cd workloads/container/
python3 generate_deploy_info.py
```

The `deploy_info.json` has the following schema:
```console
{
    vswarm-function-name:
        {
            YamlLocation: /path/to/yaml
            PredeploymentPath: [/path/to/predeployment-database/yaml]
        }
}
```

## The `PredeploymentPath` is the path to the YAML file, which is applied via `kubectl apply -f`, before creating the service under `YamlLocation`. This pre-deployment step is required in some vSwarm benchmarks, like `cartservice` which depends on a separate service `online-shop-database` before it can be started.

Similarly, the `workloads/container/generate_all_yamls.py` is a wrapper script that calls the `generate-yamls.py` script for each vSwarm benchmark in the `vSwarm_deploy_metadata.tar.gz`. The `generate-yamls.py` Python script parameterize the YAML script for the benchmark for different configurations, and creates YAML files accordingly. 

While the YAMLs are pre-generated inside the `vSwarm_deploy_metadata.tar.gz` tarball, to regenerate the YAMLs, run this from the root of this repository:
```console
tar -xzvf workloads/container/vSwarm_deploy_metadata.tar.gz -C workloads/container
cd workloads/container
python3 generate_all_yamls.py
```