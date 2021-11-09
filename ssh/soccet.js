window.addEventListener("load", function(){
    this.alert("load")
    var form2 = document.getElementById("form2")
    connectBut = this.document.getElementById("connect")
    connectBut.addEventListener("click", function() {
        connectBut.style.display = "none"
        var socket = new WebSocket("ws://127.0.0.1:6060/ws");
        socket.onopen = function() {
            document.getElementById("form-container2").style.display = "block"
          };
          
          socket.onclose = function(event) {
            if (event.wasClean) {
              alert('Соединение закрыто чисто');
            } else {
              alert('Обрыв соединения'); 
            }
            alert('Код: ' + event.code + ' причина: ' + event.reason);
            connectBut.style.display = "inline"
            document.getElementById("form-container2").style.display = "none"
          };
          
          socket.onmessage = function(event) {
            let answer = document.getElementById("answer")
            answer.innerHTML = event.data
          };
          
          socket.onerror = function(error) {
            alert("Ошибка " + error.message);
          };
          let command = document.getElementById("command")
          command.value = ""
          form2.addEventListener("submit", function(event){
            event.preventDefault();
            socket.send(command.value)
          })
    })
})  

