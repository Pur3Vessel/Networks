window.addEventListener("load", function(){
    var form = document.getElementById("form")
    let command = document.getElementById("command")
    command.value = ""
    form.addEventListener("submit", function(event){
        event.preventDefault();
        let answer = document.getElementById("answer")
        answer.innerHTML = command.value
    })
})  