## CS733 - Assignmet 4 : Putting all together.
#**FILE SERVER Using RAFT**

This is **distributed file server** implentation using **raft consensus algoritm**.
Raft cluster consists of **five servers** which replicates all the requests from client. So it can work if two of the five server went down. This also **_guarantees the consistancy_** among the cluster.

**Server** is basically a client handlers to which client can connect using <IP:Port>. Client handles also starts *Raft Node* and *State Machine* in backgroud.

**Config Folder** consists the configuration files(json). All the startup details like IP:Port of *client handle* (for communicate between file server and client) and *raft node* (for cluster communication). It also has other information like *log direcory* (stores log), *state directory* (stores persistent states)

###Package Description -
      1. fs - Consist File Server, to process client's requset(cmd).
      2. raft - Consist Raft Node, for handling communication among cluster.
      3. sm - Consist Raft State Machine, to process messages recieved by raft node from client and other sever of the cluster.
      4. main - Consist Client Handler, to serve client's request.
      
###Operating Instructions -
####  - Running Program :
          1. Run SERVER : go run raft_sm.go <server id>
          2. Run Test Cases : go test

### Things to be done -
  * Proper **synchronization** by removing all data race conditions.
  * **Garbage Collector** : For this implentation has two log one in memory and other on disk. Log is memory to *improve processing speed*. This lod is slice type, so it can be easily taken out from memory by slicing very old data. This will improve memory utilization also.  
  * **Server Restore**: Implementation is done some of manual testing is working. But proper gauranteed automate testing is to be done.
         
### REFERANCE CODE -
  * [Cluster Implementation](https://github.com/cs733-iitb/cluster)
  * [Log Implementation](https://github.com/cs733-iitb/log)
  * [File Server Implementation](https://github.com/cs733-iitb/cs733/tree/master/assignment1)




