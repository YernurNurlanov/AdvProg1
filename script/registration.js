function registerUser(event) {
	event.preventDefault();
	const email = document.getElementById("login1").value
	const password = document.getElementById("password1").value
	fetch(`/createUser?email=${email}&password=${password}`, {
			method: 'POST'
	}).then(response => {
			if (response.ok) {
					return response.json();
			} else {
					throw new Error('Network response was not ok.');
			}
	}).then(data => {
			if(data.error) {
				alert(data.error);
			}
			else if (data.success){
				alert(data.success);
				window.location.href = "/";
			}
	}).catch(error => {
			console.error('There was a problem with the fetch operation:', error);
	});
}
function loginUser(event) {
	event.preventDefault();
	const email = document.getElementById("login2").value
	const password = document.getElementById("password2").value
	fetch(`/login?email=${email}&password=${password}`, {
			method: 'POST'
	}).then(response => {
			if (response.ok) {
					return response.json();
			} else {
					throw new Error('Network response was not ok.');
			}
	}).then(data => {
			if(data.error) {
				alert(data.error);
			}
			else if (data.success){
				if (data.success == "Admin") {
					window.location.href = `/admin?token=${data.token}`;
				}
				else {
					window.location.href = `/homePage?token=${data.token}`;
				}
			}
	}).catch(error => {
			console.error('There was a problem with the fetch operation:', error);
	});
}
function showTab(tabId) {
document.querySelectorAll('.form-container').forEach(function(tab) {
		tab.style.display = 'none';
});
document.querySelectorAll('.form-container').forEach(function(btn) {
		btn.classList.remove('active');
});
document.getElementById(tabId).style.display = 'block';
document.querySelector(`[onclick="showTab('${tabId}')"]`).classList.add('active');
}