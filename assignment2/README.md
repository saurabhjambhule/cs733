## CS733 - Assignmet 1
#**FILE SERVER**

###*Configuration Instructions -*
      LevelDB - [Get LevelDB](go get github.com/syndtr/goleveldb/leveldb)

###*Operating Instructions -*
####    - Running Program :
          1. Run SERVER : go run server.go
          2. Run CLIENT : go run client.go
 
####     - Client Side Menu :
      
          1. Write - Create a File, or update the file’s contents if it already exists.
                  write <filename> <numbytes> [<exptime>]\r\n
                  <content bytes>\r\n
          2. Read - Given a Filename, retrieve the corresponding file.
                  read <filename>\r\n
          3. Compare & Swap - This replaces the old file contents with the new content provided the version is still the same.
                  cas <filename> <version> <numbytes> [<exptime>]\r\n
                  <content bytes>\r\n
          4. Append - This appends the old file contents with the new content provided the version changes.
                  append <filename> <bytes>
                  <content bytes>\r\n
          5. Delete - Delete File.
                  delete <filename>\r\n
  
####     - Error Messages :
          1. ERR_VERSION <version> - The contents were not updated because of a version mismatch & provides current version.
          2. ERR_FILE_NOT_FOUND - The filename doesn’t exist.
          3. ERR_CMD_ERR - The command is not formatted correctly.
          4. ERR_INTERNAL - Any other error you wish to report that is not covered by the rest.
  
### *REFERANCE CODE -*

  * [Build Web Application with Golang](https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/08.0.html)
  * [Network programming with Go](https://jan.newmarch.name/go/)
  * [Syndtr/goLevelDB](https://godoc.org/github.com/syndtr/goleveldb/leveldb)
  * *STACK OVERFLOW* 
  



