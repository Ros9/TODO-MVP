window.onload = function(){
	var activities = [];
	document.getElementById("add_activity").onclick = function(){
		var activity = document.getElementById("activity").value;
		if (activity === ""){
			alert("You can't write empty activity!");
		}
		else{
			document.getElementById("activity").value = "";
			activities.push(activity);
			draw();
		}
	}
	function draw(){
		var result = "";
		for (let i=0; i<activities.length; i++){
			result = result + activities[i] + '<input type=checkbox>' + '<br>';
			var x = document.getElementById
		}
		document.getElementById("activites").innerHTML = result;
	}
}