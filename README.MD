# Trinity
**Trinity Workflow** is a utility that allows you to create simple workflows that can be scheduled to run. It makes use of a custom operator and a custom resource definition for workflows. It relies on Kubernetes CronJobs to schedule and execute Workflows based on the cron schedule.

## Important Terms
1. **Workflow** 
    A Workflow consists of multiple tasks that are executed as jobs in a sequence. It is a custom resource definition and its schema is available under deployments/crd.yaml file.

2. **Task**
    A Task is an individual step within a workflow that will be executed in a separate pod. It also has access to the outputs of the previous task. If the task generates an artifact ,say .zip file, it is possible to use this artifact in other tasks of the same workflow.

    You can run a single inline command in a task or execute a shell script. Output of command execution and the status of the task (failed or success) is captured as a part of Workflow execution.

3. **Operator**
    This is a custom controller that keeps track of all the workflows that are getting created, updated and deleted and updates the state in Kubernetes objects like CronJobs. For instance when you create a workflow, operator will automatically create a cronjob and schedule it to run based on the schedule that was mentioned in the Workflow.

## Installation
1. Deploy the custom resource definition under **deployments/crd.yaml**.
2. Next deploy the **deployments/deployment.yaml** manifest. This will deploy a *clusterrole*,*clusterrolebinding*,*deployment* that will run a workflow controller. Make sure that the kubeconfig has sufficient permission to deploy these objects.
3. Now, you can start deploying your workflows. To begin with use the sample workflow available under **deployments/sample.yaml**.

## Inline vs Script
1. Inline accepts a command as string and an optional array of arguments for this command.
```
tasks:
  - name: task1    
    command:
      inline:
        command: "ls"
        args: ["-a"]
```
2. Script accepts multiline shell commands that will be executed as a script.
```
tasks:
  - name: task2   
    command:
      script: "#!/bin/bash\n echo hostname"
```

## Artifact Store
If you want to store a file or an artifact that you plan to use in other tasks, then you can turn on artifact store by setting **storeartifacts: true**. The default setting is **false**.
```
spec:
  schedule: "*/2 * * * *"
  storeartifacts: true
```
To store an artifact, move it to /artifacts/outgoing directory. Workflow will automatically scan for files under this directory for each task and upload the files to store. Similary if you want to use an artifact in other task, you can pick it up from /artifacts/incoming directory.

Workflow uses minio as an object storage. If storeartifacts is set to true, workflow will deploy minio and will use it to upload and download artifacts. This storage is available only during the execution of a workflow and will be removed once the workflow completes.

Check out the example **deployment/usingartifactstore.yaml**

Note: Artifact download for first task and upload for last task will be automatically skipped.

## Inputs to Task
In the current task,you can access the output of previous task using the environment variable **WF_INPUT**.

Check out the example **deployment/usinginputvar.yaml**

## Features I am working on
1. Option to use a user specified image for running tasks.
2. Decision tree.

## License
MIT License



