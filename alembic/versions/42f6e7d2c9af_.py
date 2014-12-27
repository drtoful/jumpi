"""empty message

Revision ID: 42f6e7d2c9af
Revises: 3eb477bf244d
Create Date: 2014-11-27 23:54:41.286402

"""

# revision identifiers, used by Alembic.
revision = '42f6e7d2c9af'
down_revision = '3eb477bf244d'

from alembic import op
import sqlalchemy as sa


def upgrade():
    ### commands auto generated by Alembic - please adjust! ###
    op.add_column('users', sa.Column('twofactor', sa.Boolean(), nullable=True))
    ### end Alembic commands ###


def downgrade():
    ### commands auto generated by Alembic - please adjust! ###
    op.drop_column('users', 'twofactor')
    ### end Alembic commands ###