## CS733 - Assignmet 1
#**FILE SERVER**

###*Configuration Instructions -*
      LevelDB - [Get LevelDB](go get github.com/syndtr/goleveldb/leveldb)

###*Operating Instructions -*
####    - Running Program :
            1. Run SERVER : go run raft_sm.go
            2. Run Test Cases : go test
 
####     - Input Events to SM :
      
            1. Append(data:[]byte): This is a request from the layer above to append the data to the replicated log. The response is in the form of an eventual Commit action (see next section).
            2. Timeout : A timeout event is interpreted according to the state. If the state machine is a leader, it is interpreted as a heartbeat timeout, and if it is a follower or candidate, it is interpreted as an election timeout.
            3. AppendEntriesReq: Message from another Raft state machine. For this and the next three event types, the parameters to the messages can be taken from the Raft paper.
            4. AppendEntriesResp: Response from another Raft state machine in response to a previous AppendEntriesReq.
            5. VoteReq: Message from another Raft state machine to request votes for its candidature.
            6. VoteResp: Response to a Vote request.

####     - Output Events by SM :
            1. Send(peerId, event). Send this event to a remote node. The event is one of AppendEntriesReq/Resp or VoteReq/Resp. Clearly the state machine needs to have some information about its peer node ids and its own id.
            2. Commit(index, data, err): Deliver this commit event (index + data) or report an error (data + err) to the layer above. This is the response to the Append event.
            3. Alarm(t): Send a Timeout after t milliseconds.
            4. LogStore(index, data []byte): This is an indication to the node to store the data at the given index. Note that data here can refer to the client’s
            
####     - Error Messages :
            1. I am not Leader - Append requested to follower instead of leader.
         
### *REFERANCE CODE -*

  * [Raft White Paper](https://www.google.co.in/url?sa=t&rct=j&q=&esrc=s&source=web&cd=4&ved=0ahUKEwiR0eXE8obLAhWEkpQKHcDoDC8QFggxMAM&url=https%3A%2F%2Framcloud.stanford.edu%2Fraft.pdf&usg=AFQjCNE8XQb0VEwFmg-Xo5yUdZpYq7BEOg&sig2=gg3NMsReCvaVK6x3hjT1CA)
  * [The Raft Consensus Algorithm](https://raft.github.io)
  * *STACK OVERFLOW* 
  



