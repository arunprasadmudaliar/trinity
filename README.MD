# Trinity
**Trinity Workflow** is a utility that allows you to create simple workflows that can be scheduled to run. It makes use of a custom operator and a custom resource definition for workflows. It relies on Kubernetes CronJobs to schedule and execute Workflows based on the cron schedule.

## Important Terms
1. **Workflow** 
    A Workflow consists of multiple tasks that are executed as jobs in a sequence. It is a custom resource definition and its schema is available under deployments/crd.yaml file.

2. **Task**
    A Task is an individual step within a workflow that will be executed in a separate pod. It also has access to the outputs of the previous task. If the task generates an artifact ,say .zip file, it is possible to use this artifact in other tasks of the same workflow.

    You can run a single inline command in a task or execute a shell script. Output of command execution and the status of the task (failed or success) is captured as a part of Workflow execution.

3. **Operator**
    This is a custom controller that keeps track of all the workflows that are getting created, updated and deleted and updates the state in Kubernetes objects like CronJobs. Workflow relies on Kubernetes CronJobs.

## Usage
1. Deploy the custom resource definition under deployments/crd.yaml.
2. Next deploy the deployments/deployment.yaml manifest. This will deploy a *clusterrole*,*clusterrolebinding*,*deployment* that will run a workflow controller. Make sure that the kubeconfig has sufficient permission to deploy these objects.
3. Now, you can start deploying your workflows. To begin with use the sample workflow available under deployments/sample.yaml.


