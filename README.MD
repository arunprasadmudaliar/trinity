# Trinity
**Trinity Workflow** is a utility that allows you to create simple workflows that can be scheduled to run. It makes use of a custom operator and a custom resource definition for workflows. It relies on Kubernetes CronJobs to schedule and execute Workflows based on the cron schedule.

## Important Terms
1. **Workflow**     
    A Workflow consists of multiple tasks that are executed as jobs in a sequence. It is a custom resource definition and its schema is available under deployments/crd.yaml file.

2. **Task**      
    A Task is an individual step within a workflow that will be executed in a separate pod. It also has access to the outputs of the previous task. If the task generates an artifact ,say .zip file, it is possible to use this artifact in other tasks of the same workflow.

    You can run a single inline command in a task or execute a shell script. Output of command execution and the status of the task (failed or success) is captured as a part of Workflow execution.
3. **Run**     
    The execution status of a Workflow and its tasks are updated under status field of the Workflow. An execution of a Workflow is called as a **RUN**.

3. **Operator**    
    This is a custom controller that keeps track of all the workflows that are getting created, updated and deleted and updates the state in Kubernetes objects like CronJobs. For instance when you create a workflow, operator will automatically create a cronjob and schedule it to run based on the schedule that was mentioned in the Workflow.

## Installation
1. Deploy the custom resource definition under **deployments/crd.yaml**.
2. Next deploy the **deployments/deployment.yaml** manifest. This will deploy a *clusterrole*,*clusterrolebinding*,*deployment* that will run a workflow controller. Make sure that the kubeconfig has sufficient permission to deploy these objects.
3. Now, you can start deploying your workflows. To begin with use the sample workflow available under **examples/basic.yaml**.

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

Workflow uses minio as an object storage. If *storeartifacts* is set to *true*, workflow will deploy minio and will use it to upload and download artifacts. This storage is available only during the execution of a workflow and will be removed once the workflow completes.

Check out the example **examples/usingartifactstore.yaml**

Note: Artifact download for first task and upload for last task will be automatically skipped.

## Accessing output of previous task
In the current task,you can access the output of previous task using the environment variable **WF_INPUT**.

Check out the example **examples/usinginputvar.yaml**

## Track the execution status of Workflow and its tasks
The status of a Workflow is available under the status field of the workflow. You can use ```kubectl describe workflow <workflow-name>``` or ```kubectl get workflow <workflow-name> -o json``` to view the results of the execution. Workflow will maintain the results of all the executions under **RUNS**.
```
"status": {
                "runs": [
                    {
                        "ended_at": "02-14-2021 10:42:21",
                        "id": 1,
                        "phase": "completed",
                        "started_at": "02-14-2021 10:42:08",
                        "tasks": [
                            {
                                "error": "",
                                "name": "task1",
                                "output": "",
                                "status": "success"
                            },
                            {
                                "error": "",
                                "name": "task2",
                                "output": "Hello\n",
                                "status": "success"
                            }
                        ]
                    },
```

## Features I am working on
1. Option to use a user specified image for running tasks.
2. Decision tree.

## Contribution
A contribution to this project is welcome through a pull request. Moreover, I am not a full time golang developer and hence the code base might not be idiomatic. Any kind of help will be highly appreciated.

## License
MIT License



