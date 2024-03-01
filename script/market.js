function homePage(token) {
	window.location.href = `/homePage?token=${token}`;
}
// move card to collection
function sendCardData(cardId,userId) {
	fetch(`/cardToCollection?user_id=${userId}&card_id=${cardId}`, {
		method: 'POST'
	})
	.then(response => response.json())
	.then(data => {
			if(data.error) { alert(data.error); }
			else { alert(data.success); }
	})
	.catch(error => {
			console.error('There was a problem with the fetch operation:', error);
	});
}
// display cards
let currentPage = 1
var page = document.getElementById("currentPage")
let filter = document.getElementById('positionFilter').value
let sortBy = document.getElementById('sortCards').value
document.getElementById('prevPage').disabled = currentPage === 1;
document.addEventListener('DOMContentLoaded', fetchData);
function fetchData() {
		fetch(`/marketCards?page=${currentPage}&filter=${encodeURIComponent(filter)}&sort=${encodeURIComponent(`card_id.${sortBy}`)}`)
				.then(response => response.json())
				.then(data => {
						if(data.error) { alert(data.error); }
						else { displayData(data); }
				})
				.catch(error => {
						console.error('There was a problem with the fetch operation:', error);
				});
}
function displayData(data) {
		const dataContainer = document.getElementById('marketCards');
		
		data.forEach(item => {

					const user_id = document.createElement('p');
					user_id.textContent = `User_id: ${item["user_id"]}`;

					const div = document.createElement('div');
					div.classList.add('card');

					const card_id = document.createElement('p');
					card_id.textContent = `Card_id: ${item["card_id"]["card_id"]}`;

					const name = document.createElement('p');
					name.textContent = `Name: ${item["card_id"]["name"]}`;

					const nationality = document.createElement('p');
					nationality.textContent = `Nationality: ${item["card_id"]["nationality"]}`;

					const club = document.createElement('p');
					club.textContent = `Club: ${item["card_id"]["club"]}`;

					const position = document.createElement('p');
					position.textContent = `Position: ${item["card_id"]["position"]}`;
					
					div.appendChild(user_id);
					div.appendChild(card_id);
					div.appendChild(name);
					div.appendChild(club);
					div.appendChild(nationality);
					div.appendChild(position);
					dataContainer.appendChild(div);
		});
}
// filtering
document.getElementById("filterForm").addEventListener("submit", function(event){
	event.preventDefault();
	filter = document.getElementById('positionFilter').value;
	if (filter) {
		currentPage = 1
		page.innerHTML = currentPage;
		document.getElementById('nextPage').disabled = false;
		document.getElementById('prevPage').disabled = currentPage === 1;
		fetch(`/marketCards?page=${currentPage}&filter=${encodeURIComponent(filter)}&sort=${encodeURIComponent(`card_id.${sortBy}`)}`, {
				method: 'GET',
		})
		.then(response => response.json())
		.then(data => {
				if(data.error) { alert(data.error); }
				else { 
					var myDiv = document.getElementById('marketCards');
					myDiv.innerHTML = '';
					displayData(data); 
				}
		})
		.catch(error => {
				console.error('There was a problem with the fetch operation:', error);
		});
	} else {
				alert('Please select a position before submitting.');
		}
})
// sorting
document.getElementById("sortForm").addEventListener("submit", function(event){
	event.preventDefault();
	sortBy = document.getElementById('sortCards').value;
	filter = document.getElementById('positionFilter').value;
	if (sortBy) {
		currentPage = 1
		page.innerHTML = currentPage;
		document.getElementById('nextPage').disabled = false;
		document.getElementById('prevPage').disabled = currentPage === 1;
		fetch(`/marketCards?page=${currentPage}&filter=${encodeURIComponent(filter)}&sort=${encodeURIComponent(`card_id.${sortBy}`)}`, {
				method: 'GET',
		})
		.then(response => response.json())
		.then(data => {
				if(data.error) { alert(data.error); }
				else { 
					var myDiv = document.getElementById('marketCards');
					myDiv.innerHTML = '';
					displayData(data); 
				}
		})
		.catch(error => {
				console.error('There was a problem with the fetch operation:', error);
		});
	} else {
				alert('Please select a position before submitting.');
		}
})
// pagination
document.getElementById("nextPage").addEventListener("click", function(){
	currentPage += 1
	page.innerHTML = currentPage;
	fetch(`/marketCards?page=${currentPage}&filter=${encodeURIComponent(filter)}&sort=${encodeURIComponent(`card_id.${sortBy}`)}`, {
		method: 'GET',
	})
		.then(response => response.json())
		.then(data => {
				if(data.error) { alert(data.error); }
				else { 
					var myDiv = document.getElementById('marketCards');
					myDiv.innerHTML = '';
					displayData(data); 
				}
		})
		.catch(error => {
				console.error('There was a problem with the fetch operation:', error);
		});
	document.getElementById('prevPage').disabled = currentPage === 1;
	if (currentPage != 1) {
		document.getElementById('prevPage').disabled = false;
	}
	document.getElementById('nextPage').disabled = currentPage === 8;
	if (currentPage != 8) {
		document.getElementById('nextPage').disabled = false;
	}
})
document.getElementById("prevPage").addEventListener("click", function(){
	currentPage -= 1;
	page.innerHTML = currentPage;
	fetch(`/marketCards?page=${currentPage}&filter=${encodeURIComponent(filter)}&sort=${encodeURIComponent(`card_id.${sortBy}`)}`, {
		method: 'GET',
	})
		.then(response => response.json())
		.then(data => {
				if(data.error) { alert(data.error); }
				else { 
					var myDiv = document.getElementById('marketCards');
					myDiv.innerHTML = '';
					displayData(data); 
				}
		})
		.catch(error => {
				console.error('There was a problem with the fetch operation:', error);
		});
	document.getElementById('prevPage').disabled = currentPage === 1;
	if (currentPage != 1) {
		document.getElementById('prevPage').disabled = false;
	}
	document.getElementById('nextPage').disabled = currentPage === 8;
	if (currentPage != 8) {
		document.getElementById('nextPage').disabled = false;
	}
})