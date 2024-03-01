function homePage(token) {
	window.location.href = `/homePage?token=${token}`;
}
var divElement = document.querySelector('.card-collection');
var userId = divElement.getAttribute('data-user-id');
document.addEventListener('DOMContentLoaded', fetchPlayersData);
// saved card to zones
function fetchPlayersData() {
	fetch(`/getTeam?user_id=${userId}`,{
		method: "POST"
	})
	.then(response => response.json())
	.then(data => {
			if(data.error) { alert(data.error); }
			else { placePlayersInZones(data); }
	})
	.catch(error => {
			console.error('There was a problem with the fetch operation:', error);
	});
}
// Save team button click event
document.getElementById('saveTeamButton').addEventListener('click', () => {
		const dropZones = document.querySelectorAll('.drop-zone');
		const teamData = [];

		dropZones.forEach(zone => {
				const zoneId = zone.id;
				const players = zone.querySelectorAll('.card');
				players.forEach(player => {
						const playerId = player.dataset.playerId;
						teamData.push({ playerId, position: zoneId });
				});
		});
		fetch(`/saveTeam?user_id=${userId}`, {
				method: 'POST',
				headers: {
						'Content-Type': 'application/json'
				},
				body: JSON.stringify(teamData)
		})
		.then(response => response.json())
		.then(data => {
				if(data.error) { alert(data.error); }
				else if (data.success){ alert(data.success); }
		})
		.catch(error => {
				console.error('There was a problem with the fetch operation:', error);
		});
});
// move card to zone
document.addEventListener('DOMContentLoaded', () => {
	const cards = document.querySelectorAll('.card');
	let draggedCard = null;
	// Enable dragging for each card
	cards.forEach(card => {
			card.addEventListener('dragstart', (e) => {
					draggedCard = card;
					setTimeout(() => card.classList.add('dragging'), 0);
			});
			card.addEventListener('dragend', () => {
					draggedCard = null;
					card.classList.remove('dragging');
			});
	});
	const dropZones = document.querySelectorAll('.drop-zone');
	// Enable each drop zone to accept cards
	dropZones.forEach(zone => {
			zone.addEventListener('dragover', (e) => {
					e.preventDefault(); // Necessary to allow dropping
					zone.classList.add('hover');
			});
			zone.addEventListener('dragleave', () => {
					zone.classList.remove('hover');
			});
			zone.addEventListener('drop', (e) => {
					e.preventDefault();
					if (draggedCard) { // Check if a card is being dragged
							const zoneId = zone.id; // Получаем ID зоны
							const draggedPosition = draggedCard.querySelector('p:nth-child(2)').textContent.trim(); // Получаем позицию игрока
							// Преобразуем ID зоны в соответствующую строку позиции
							let zonePosition = '';
							if (zoneId === 'leftForward' || zoneId === 'centerForward' || zoneId === 'rightForward') {
									zonePosition = 'Forward';
							} else if (zoneId === 'leftMidfielder' || zoneId === 'centerMidfielder' || zoneId === 'rightMidfielder') {
									zonePosition = 'Midfielder';
							} else if (zoneId === 'leftDefender' || zoneId === 'leftCenterDefender' || zoneId === 'rightCenterDefender' || zoneId === 'rightDefender') {
									zonePosition = 'Defender';
							} else if (zoneId === 'goalkeeper') {
									zonePosition = 'Goalkeeper';
							} else if (zoneId === 'manager') {
									zonePosition = 'Manager';
							}
							if (draggedPosition === zonePosition) { // Проверяем, соответствует ли позиция игрока позиции зоны
									const existingCards = zone.querySelectorAll('.card'); // Получаем все карты в зоне
									if (existingCards.length === 0) { // Если в зоне нет других карт
											zone.appendChild(draggedCard); // Добавляем карту в новую зону
									} else {
											// Получаем контейнер для карт
											const cardContainer = document.querySelector('.card-collection .cards');
											// Получаем первую карту в контейнере
											cardContainer.appendChild(existingCards[0]);
									}
							}
							zone.classList.remove('hover');
					}
			});
	});
	// Enable each card to be dragged back to the card collection
	cards.forEach(card => {
			card.addEventListener('dragover', (e) => {
					e.preventDefault(); // Necessary to allow dropping
					card.classList.add('hover');
			});

			card.addEventListener('dragleave', () => {
					card.classList.remove('hover');
			});

			card.addEventListener('drop', (e) => {
					e.preventDefault();
					if (draggedCard) { // Check if a card is being dragged
							card.parentNode.insertBefore(draggedCard, card);
							card.classList.remove('hover');
					}
			});
	});
});

function placePlayersInZones(data) {
	data.forEach(playerData => {
			const { playerId, position } = playerData;
			const playerCard = document.querySelector(`.card[data-player-id="${playerId}"]`);
			const zone = document.getElementById(position);

			if (playerCard && zone) {
					zone.appendChild(playerCard);
			}
	});
}
