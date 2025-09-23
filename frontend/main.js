// textInput = document.getElementById("textInput");
// textToType = document.getElementById("textToType");

// textInput.addEventListener("input", function (event) {
//   console.log("Event:", event);
// });

htmx.config.wsReconnectDelay = function (retryCount) {
  return retryCount * 1000; // return value in milliseconds
};
