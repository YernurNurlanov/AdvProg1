function getValue(name) {
	let cookieArr = document.cookie.split(";");
	for(let i = 0; i < cookieArr.length; i++) {
		let cookiePair = cookieArr[i].split("=");
		if(name == cookiePair[0].trim()) {
			return decodeURIComponent(cookiePair[1]);
		}
	}
	return null;
}
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
function showChats(tabId) {
	document.querySelectorAll('.form-container').forEach(function(tab) {
		tab.style.display = 'none';
	});
	document.querySelectorAll('.form-container').forEach(function(btn) {
			btn.classList.remove('active');
	});
	document.getElementById(tabId).style.display = 'block';
	document.querySelector(`[onclick="showChats('${tabId}')"]`).classList.add('active');
	fetch(`/${tabId}?id=${getValue("user-data")}`)
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
				else {
					const container = document.getElementById(tabId);
					container.innerHTML = '';
					data.chats.forEach(chat => {
							const chatLabel = document.createElement('label');
							chatLabel.className = 'chat-label';
							const radioBtn = document.createElement('input');
							radioBtn.type = 'radio';
							radioBtn.name = 'chatId';
							radioBtn.value = chat.ID;
							const textNode = document.createTextNode(` ID: ${chat.ID}, ClientID: ${chat.Id_client}`);
							chatLabel.appendChild(radioBtn);
							chatLabel.appendChild(textNode);
							container.appendChild(chatLabel);
					});
					const sendBtn = document.createElement('button');
					sendBtn.textContent = 'Get Selected Chat';
					sendBtn.addEventListener('click', () => {
						const selectedChatId = document.querySelector('input[name="chatId"]:checked');
						if (!selectedChatId) {
								alert('Please select a chat before pressing the button.');
								return;
						}
						openChatModal(selectedChatId.value);
					});
					container.appendChild(sendBtn);
				}
		}).catch(error => {
				console.error('Error fetching chats:', error);
		});
}
function openChatModal(chatId) {
	var adminWs
	try {
		adminWs = new WebSocket(`ws://localhost:8080/handleAdmin?id=${getValue("user-data")}&chat_id=${chatId}`);
	} 
	catch (error) {
			alert('Failed to connect to the WebSocket server.');
	}
	const modal = document.getElementById('chatModal');
	const modalMessagesDiv = document.getElementById('modalMessages');
	modalMessagesDiv.innerHTML = `<p>Chat ID: ${chatId}</p>`;
	modal.style.display = "block";
	fetch(`/getChat?id=${getValue("user-data")}&role=admin`)
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
		else {
			const modalMessagesDiv = document.getElementById('modalMessages');
			for (i = 0; i < data.length; i++){
				modalMessagesDiv.innerHTML += `<p>${data[i].Role}: ${data[i].MessageText}</p>`;
			}
			
		}
	}).catch(error => {
			console.error('Error fetching chats:', error);
	});
	const span = document.getElementsByClassName("close")[0];
	span.onclick = () => {
			modal.style.display = "none";
	};
	window.onclick = (event) => {
			if (event.target == modal) {
					modal.style.display = "none";
			}
	};
	adminWs.onmessage = (event) => {
    const modalMessagesDiv = document.getElementById('modalMessages');
    modalMessagesDiv.innerHTML += `<p>user: ${event.data}</p>`;
	};
	document.getElementById('modalSendButton').onclick = () => {
			const input = document.getElementById('modalMessageInput');
			const message = input.value;
			adminWs.send(message);
			input.value = '';
			modalMessagesDiv.innerHTML += `<p>admin: ${message}</p>`;
	};
	document.getElementById('modalCloseButton').onclick = () => {
		closeChat(adminWs);
		modal.style.display = "none";
		location.reload();
	};
}
function closeChat(adminWs) {
	if (adminWs && adminWs.readyState === WebSocket.OPEN) {
			adminWs.send("close");
	} else {
			console.error("WebSocket connection is not open.");
	}
}
function showMessages(){
	fetch(`/getChat?id=${getValue("user-data")}`)
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
	else {
		const modalMessagesDiv = document.getElementById('modalMessages');
		for (i = 0; i < data.length; i++){
			modalMessagesDiv.innerHTML += `<p>${data[i].Role}: ${data[i].MessageText}</p>`;
		}
		
	}
}).catch(error => {
		console.error('Error fetching chats:', error);
});
}
