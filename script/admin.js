const userCheckboxes = document.querySelectorAll('.user-checkbox');
userCheckboxes.forEach(function(checkbox) {
	checkbox.addEventListener('change', function() {
		userCheckboxes.forEach(function(cb) {
			if (cb !== checkbox) {
					cb.checked = false;
			}
		});
	});
});
function deleteUser(event) {
	event.preventDefault();
	const selectedUserId = document.querySelector('.user-checkbox:checked').value;
	fetch(`/deleteUser?id=${selectedUserId}`, {
		method: 'DELETE'
	})
	.then(response => {
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
			location.reload();
		}
	}).catch(error => {
		console.error('There was a problem with the fetch operation:', error);
	});
}
function addQuestions(event) {
	event.preventDefault();
	var question = document.getElementById('question').value;
	var option1 = document.getElementById('option1').value;
	var option2 = document.getElementById('option2').value;
	var option3 = document.getElementById('option3').value;
	var option4 = document.getElementById('option4').value;
	var rightOp = document.getElementById('rightOp').value;
	var formData = {
		Question: question,
		OptionA: option1,
		OptionB: option2,
		OptionC: option3,
		OptionD: option4,
		RightOp: rightOp,
	};
	fetch('/addQuestion', {
		method: 'POST',
		headers: {
				'Content-Type': 'application/json'
		},
		body: JSON.stringify(formData)
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
				location.reload();
			}
	}).catch(error => {
			console.error('There was a problem with the fetch operation:', error);
	});
}
function addCards(event) {
	event.preventDefault();
	var name = document.getElementById('name').value;
	var nationality = document.getElementById('nationality').value;
	var club = document.getElementById('club').value;
	var position = document.getElementById('position').value;
	var formData = {
			Name: name,
			Nationality: nationality,
			Club: club,
			Position: position
	};
	fetch('/addCard', {
		method: 'POST',
		headers: {
				'Content-Type': 'application/json'
		},
		body: JSON.stringify(formData)
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
				location.reload();
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