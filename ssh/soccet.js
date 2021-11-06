window.addEventListener("load", function(){
    this.alert("load")
  
    var form2 = document.getElementById("form2")
    connectBut = this.document.getElementById("connect")
    connectBut.addEventListener("click", function() {
        var socket = new WebSocket("ws://127.0.0.1:6060/ws");
        socket.onopen = function() {
            alert("Соединение установлено.");
            document.getElementById("form-container2").style.display = "block"
          };
          
          socket.onclose = function(event) {
            if (event.wasClean) {
              alert('Соединение закрыто чисто');
            } else {
              alert('Обрыв соединения'); // например, "убит" процесс сервера
            }
            alert('Код: ' + event.code + ' причина: ' + event.reason);
          };
          
          socket.onmessage = function(event) {
            alert("Получены данные " + event.data);
          };
          
          socket.onerror = function(error) {
            alert("Ошибка " + error.message);
          };
    })
    let command = document.getElementById("command")
    command.value = ""
    form2.addEventListener("submit", function(event){
        event.preventDefault();
        let answer = document.getElementById("answer")
        answer.innerHTML = command.value
    })
})  