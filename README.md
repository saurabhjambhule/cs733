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
          2. Read - Given a Filename, retrieve the corresponding file.
          3. Compare & Swap - This replaces the old file contents with the new content provided the version is still the same.
          4. Append - This appends the old file contents with the new content provided the version changes.
          5. Rename - Rename File.
          6. Delete - Delete File.
  
  
####     - Server Side Menu :
          1. Sigle Client - Serves only single client at a time.
          2. Multi Client - Supports multiple client request.
  
####     - Error Messages :
          1. ERR_VERSION - The contents were not updated because of a version mismatch.
          2. ERR_FILE_NOT_FOUND - The filename doesn’t exist.
          3. ERR_CMD_ERR - The command is not formatted correctly.
          4. ERR_INTERNAL - Any other error you wish to report that is not covered by the rest.

### *INCOMPLETE CODE -*
  
  * Multi Client - partialy runs.
  * Expire Time - not done.
  * Muli-version - not done.
  * Rename - not done.
  * Delete - not working.
  * Code Documentation - not done.
  * Test Case - not done.
  
### *REFERANCE CODE -*

  * [Build Web Application with Golang](https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/08.0.html)
  * [Network programming with Go](https://jan.newmarch.name/go/)
  * [Syndtr/goLevelDB](https://godoc.org/github.com/syndtr/goleveldb/leveldb)
  * *STACK OVERFLOW* 
  



